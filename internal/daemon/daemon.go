package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// codegraphDirName / storeDirName mirror internal/cli's constants of the
// same name (internal/cli/init.go) — duplicated locally rather than
// imported to avoid an internal/daemon -> internal/cli dependency edge:
// cli is the outermost layer that will depend on daemon (Plan 04-08's
// `codegraph daemon` command), not the other way around. This is the same
// cross-package-duplication precedent internal/query set for its own
// codegraphDirName (Phase 3) and internal/indexer.ShouldSkipDir.
const (
	codegraphDirName = ".codegraph"
	storeDirName     = "store"
)

// staleSidecarName mirrors internal/query/status.go's staleSidecarName
// (".sync-pending", D-04a) — duplicated locally per the same precedent:
// internal/query is the READ side (Status/Explore check for its
// presence); internal/daemon is the WRITE side (touch on the first
// pending event, remove on a successful commit).
const staleSidecarName = ".sync-pending"

// ErrNotInitialized mirrors the internal/cli and internal/query sentinels
// of the same name — returned by New when repoRoot has no .codegraph/ yet.
var ErrNotInitialized = errors.New("daemon: not initialized")

// ErrWatcherClosed is returned by Run when the watch loop exits without ctx
// being cancelled — fsnotify's Events/Errors channels closed abnormally
// (03-REVIEW.md IN-07). Without this, Run would keep blocking on
// <-ctx.Done() holding the daemon lockfile with no watcher running: a
// silent zombie lock-holder every other session's RunWithRetry defers to
// forever (pid alive, so isStale never clears it) while the graph silently
// stops auto-updating. It is neither ErrLockLive nor watch.ErrWatchDisabled,
// so RunWithRetry surfaces it immediately and serve's watcher goroutine
// logs it to stderr.
var ErrWatcherClosed = errors.New("daemon: watcher event stream closed unexpectedly")

// flushRetryPath is the sentinel path Run's requeue wrapper feeds back into
// the Debouncer when a flush's indexer.Sync lost a Pebble LOCK race
// (03-REVIEW.md CR-01 scenario 2: an in-flight codegraph_explore held the
// store open when the debounce fired). indexer.Sync re-diffs the whole repo
// regardless of the coalesced path set, so the sentinel's value is
// irrelevant — it exists only to re-arm the timer, because the Debouncer
// otherwise only fires on organic watcher events: without the requeue, a
// failed flush after the edit burst ends would strand the .sync-pending
// sidecar (staleness observable, content stale) until the next unrelated
// event or the next session's reconcile.
const flushRetryPath = "\x00codegraph-flush-lock-retry"

// maxFlushLockRequeues bounds consecutive lock-held flush requeues (CR-01 /
// Phase-2 BL-01 lesson: bounded retries only, never an unbounded loop). The
// counter resets on any successful flush AND at exhaustion (the give-up
// branch, 03-REVIEW.md IN-03), so the budget is genuinely per-contention-
// episode, not per-session: a fresh episode hours later gets its own five
// requeues instead of inheriting zero from an old exhausted one. Once
// exhausted, the sidecar stays set and the next organic watcher event or
// session reconcile picks the work back up — the give-up branch never
// requeues, so each episode's chain stays bounded.
const maxFlushLockRequeues = 5

// Daemon is the long-lived local process (or in-process fallback, D-05)
// that owns a repo's watcher and drives every debounced flush through
// indexer.Sync — the single coordinated writer multiple agent sessions
// share (SYNC-04, INDX-05). Construct with New, then call Run(ctx); Run
// blocks until ctx is cancelled and every spawned goroutine has joined
// (D-07).
type Daemon struct {
	repoRoot     string
	codegraphDir string
	storeDir     string
	opts         indexer.Options

	// syncMu serializes indexer.Sync invocations. The debouncer already
	// guarantees at most one flush is scheduled at a time in the common
	// case, but a Sync that runs longer than the debounce window must
	// never overlap the next debounced flush firing — syncMu is the
	// belt-and-suspenders guarantee behind "exactly one GraphStore.Writer
	// at a time" (INDX-05), independent of debounce timing.
	syncMu sync.Mutex

	// onSync, when non-nil, is invoked after every Sync attempt (success
	// or failure). It is a test-only observability seam (unexported field,
	// no exported setter) so daemon_test.go can synchronize on a sync
	// actually completing instead of racing the debounce timer with
	// time.Sleep. Production callers (Plan 04-08's `codegraph daemon` /
	// serve's in-process fallback) leave it nil.
	onSync func(indexer.Stats, error)

	// onSyncStart, when non-nil, is invoked at the very start of flush,
	// before touchPending or indexer.Sync run. It is a test-only
	// control seam (mirrors onSync) that lets daemon_test.go
	// deterministically hold a flush "in flight" — CR-01's exact
	// untracked-goroutine window — long enough to prove Run's shutdown
	// path genuinely waits for it, rather than racing a real
	// indexer.Sync's typically sub-millisecond duration against a tiny
	// test fixture. Production callers leave it nil.
	onSyncStart func()

	// syncFn, when non-nil, replaces indexer.Sync inside flush. It is a
	// test-only control seam (unexported, no exported setter — mirrors
	// onSync/onSyncStart) letting daemon_test.go inject deterministic
	// Sync outcomes — specifically graphstore.ErrStoreLocked, to force
	// the requeue chain to exhaustion (WR-01) — WITHOUT real Pebble lock
	// contention: a pebble.Open that fails on a held LOCK can leak its
	// disk-health ticker goroutine (the ticker restarts after the Open
	// error path's FS Close when an op is still in flight), which trips
	// this package's goleak TestMain gate. Real ErrStoreLocked
	// propagation through indexer.Sync stays covered by the integration
	// live-sync test. Production callers leave it nil.
	syncFn func(repoRoot, storeDir string, opts indexer.Options) (indexer.Stats, error)

	// onWatchOpen, when non-nil, is invoked with the freshly-opened
	// watcher right after watch.Open succeeds inside Run, before the watch
	// loop goroutine starts. It is a test-only control seam (unexported,
	// no exported setter — mirrors onSyncStart) so daemon_test.go can
	// capture the watcher handle and close its event stream out from under
	// a running Run, deterministically driving the abnormal-teardown path
	// that must return ErrWatcherClosed (03-REVIEW.md IN-07/WR-01) —
	// fsnotify offers no other way to force its channels closed without
	// the ctx being cancelled. Production callers leave it nil.
	onWatchOpen func(*watch.Watcher)

	// probe carries the flag-derived watch.Probe inputs (D-01..D-04's
	// NoWatch/ForceWatch) that Run's policy gate (WATCH-03/D-11) checks
	// before ever touching the lockfile. Env/IsWSL are left nil here so
	// they default inside watch.WatchDisabledReason to os.Getenv/DetectWSL
	// — the standalone `codegraph daemon` command (internal/cli/daemon.go,
	// UNCHANGED by this plan) never sets probe, so real env vars and real
	// WSL detection still apply for that path. serve --mcp (03-03) sets
	// probe via WithProbe to carry its own --no-watch/--watch flags.
	probe watch.Probe
}

// Option customizes a Daemon constructed via New. The variadic parameter
// keeps New's existing two-argument call sites (internal/cli/daemon.go)
// source- and binary-compatible — passing zero Options is a no-op.
type Option func(*Daemon)

// WithProbe overrides the Daemon's watch.Probe (see the probe field's doc
// comment). This is the only Option this plan introduces.
func WithProbe(p watch.Probe) Option {
	return func(d *Daemon) {
		d.probe = p
	}
}

// New resolves repoRoot's .codegraph/ layout and returns a Daemon ready to
// Run. It does not touch the lockfile or open the store — those happen
// inside Run, so a not-yet-started Daemon never holds any process-wide
// resource. New fails with ErrNotInitialized if repoRoot has no
// .codegraph/ (mirrors internal/cli's index/sync guidance: run `codegraph
// init` first). opts (WR-04) is threaded through to every debounced
// indexer.Sync call this Daemon drives (flush, below) — e.g. Workers to
// bound the daemon's own extraction pool independently of the CLI's
// one-shot `codegraph sync`/`index` invocations.
func New(repoRoot string, opts indexer.Options, options ...Option) (*Daemon, error) {
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	codegraphDir := filepath.Join(abs, codegraphDirName)
	if _, err := os.Stat(codegraphDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s does not exist — run `codegraph init` first", ErrNotInitialized, codegraphDir)
		}
		return nil, err
	}
	d := &Daemon{
		repoRoot:     abs,
		codegraphDir: codegraphDir,
		storeDir:     filepath.Join(codegraphDir, storeDirName),
		opts:         opts,
	}
	for _, opt := range options {
		opt(d)
	}
	return d, nil
}

// Run acquires the daemon lockfile (single-writer invariant, D-05), opens
// a recursive watcher over repoRoot, and drives every debounced flush
// through indexer.Sync — touching the .sync-pending sidecar on the first
// pending event and removing it on a successful commit (D-04a). Run
// blocks until ctx is cancelled; it releases the lock and returns only
// after the watcher goroutine it spawned has joined (sync.WaitGroup, D-07)
// AND any debounce flush already in flight — including its indexer.Sync
// call — has completed (deb.Wait(), CR-01) — no goroutine outlives Run and
// no Sync is still writing when the lock is released (SYNC-06, INDX-05). If
// another live daemon already holds the lock, Run returns ErrLockLive
// immediately without starting a watcher. If the watch loop exits without
// ctx being cancelled (abnormal fsnotify teardown, IN-07), Run tears down
// through the same join path, releases the lock, and returns
// ErrWatcherClosed instead of holding the lock as a zombie — though on
// that path (ctx still live) an in-flight lock-lost flush can extend the
// join by a bounded requeue chain; see the backstop deb.Stop() comment in
// the function body (IN-01).
//
// Before any of the above, Run enforces watch.WatchDisabledReason as its
// FIRST action (WATCH-03/D-11): a policy-disabled Daemon returns a
// watch.ErrWatchDisabled-wrapped error and never calls acquire(), so a
// disabled watcher never touches the lockfile. This is the single shared
// enforcement point both the in-process `serve --mcp` watcher (03-03) and
// the standalone `codegraph daemon` command (internal/cli/daemon.go,
// unchanged) inherit through this one call.
func (d *Daemon) Run(ctx context.Context) error {
	if reason := watch.WatchDisabledReason(d.repoRoot, d.probe); reason != "" {
		// IN-05: the typed error carries the reason so CLI consumers
		// extract it via errors.As instead of re-deriving it —
		// errors.Is(err, watch.ErrWatchDisabled) still matches.
		return &watch.DisabledError{Reason: reason}
	}
	if err := acquire(d.codegraphDir); err != nil {
		return err
	}
	defer func() {
		if err := release(d.codegraphDir); err != nil {
			log.Printf("daemon: releasing lock: %v", err)
		}
	}()

	w, err := watch.Open(d.repoRoot)
	if err != nil {
		return err
	}
	defer w.Close()

	if d.onWatchOpen != nil {
		d.onWatchOpen(w)
	}

	// CR-01 scenario 2 (03-REVIEW.md): the flush callback is wrapped so a
	// Sync that lost a Pebble LOCK race (a concurrent explore/reconcile had
	// the store open past graphstore.Open's own bounded in-call retries)
	// re-arms the debouncer instead of being silently terminal. Bounded by
	// maxFlushLockRequeues (BL-01 lesson: no unbounded loops); the ctx.Err()
	// gate keeps a cancelled shutdown from scheduling one more timer that
	// deb.Wait() below would then have to wait out, and — per the same
	// BL-01 lesson — a cancelled retry exits without recording anything:
	// flush already leaves the .sync-pending sidecar in place on every
	// failure path, so no verdict is ever persisted under a cancelled ctx.
	// deb is captured by the closure before any timer can fire: the callback
	// only runs after an Add, and Adds only start once w.Run is live below.
	var deb *watch.Debouncer
	var lockRequeues int32
	deb = watch.NewDebouncer(ctx, watch.DebounceDuration(), func(paths map[string]struct{}) {
		err := d.flush(paths)
		switch {
		case err == nil:
			atomic.StoreInt32(&lockRequeues, 0)
		case errors.Is(err, graphstore.ErrStoreLocked) && ctx.Err() == nil:
			if n := atomic.AddInt32(&lockRequeues, 1); n <= maxFlushLockRequeues {
				deb.Add(flushRetryPath)
			} else {
				// IN-03: n counts consecutive lock-lost syncs (1 organic +
				// maxFlushLockRequeues requeued), and resetting here — not
				// only on success — keeps the budget per-episode as
				// maxFlushLockRequeues' doc comment promises. The ctx.Err()
				// gate above means a cancelled shutdown never reaches this
				// reset (BL-01: nothing recorded under a cancelled ctx).
				atomic.StoreInt32(&lockRequeues, 0)
				log.Printf("daemon: sync lost the store-lock race %d consecutive times; giving up until the next event (graph stays marked stale via %s)", n, staleSidecarName)
			}
		}
	})

	// IN-07: loopExited lets Run notice the watch loop returning WITHOUT
	// ctx being cancelled (fsnotify's channels closed abnormally). Blocking
	// on <-ctx.Done() alone would leave Run holding the daemon lockfile
	// with no watcher running — a zombie lock-holder other sessions defer
	// to forever while the graph silently stales.
	loopExited := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(loopExited)
		w.Run(ctx, deb)
	}()

	var runErr error
	select {
	case <-ctx.Done():
	case <-loopExited:
		if ctx.Err() == nil {
			runErr = ErrWatcherClosed
		}
	}
	wg.Wait()
	// IN-04 belt-and-suspenders: watchLoop's own deb.Stop() ran inside
	// w.Run (joined by wg.Wait() above), but a requeue Add racing
	// cancellation can re-arm the timer after that Stop; this idempotent
	// second Stop cancels it so deb.Wait() below doesn't ride out a dead
	// debounce window. The structural fix is Debouncer.Add's own ctx gate
	// (internal/watch/debounce.go) — this is the caller-side backstop.
	//
	// IN-01 (round 5): both of those defenses are ctx gates, so they only
	// cover the CANCELLATION teardown. In the loopExited-with-live-ctx
	// path (ErrWatcherClosed above), ctx is not done: a lock-lost
	// in-flight flush can requeue PAST this Stop (Add re-arms because ctx
	// is alive) and deb.Wait() below rides out the new timer — repeatable
	// up to maxFlushLockRequeues times, so Run's return can lag by up to
	// that many debounce windows plus sync durations. Bounded and
	// invariant-safe (the chain terminates via success, non-lock error,
	// or give-up; the lock is correctly held throughout, and the extra
	// syncs are legitimate pending work), so this is accepted teardown
	// latency on an already-abnormal path — cancelling an internal
	// context here would be the fix if prompt teardown ever matters.
	deb.Stop()
	// CR-01: wg.Wait() only joins the tracked watcher goroutine — it does
	// NOT join a debounce flush that had already started running its
	// indexer.Sync (on the timer's own untracked goroutine) when ctx was
	// cancelled. deb.Stop() (watchLoop's and the backstop above) can
	// only cancel a timer that hasn't fired yet; deb.Wait() is the
	// explicit join for one that has, closing the window where Run
	// released the daemon lock (see the deferred release() above) while a
	// Sync was still mid-commit against the single coordinated Writer
	// (INDX-05, D-07, SYNC-06).
	deb.Wait()
	return runErr
}

// RunWithRetry drives d.Run(ctx) in a loop, converging concurrent
// serve --mcp sessions on a single writer (WATCH-04/D-14): defer-and-retry
// replaces the prior defer-once behavior, so a session that lost the race
// for the lock does not give up forever — it retries on a jittered cadence
// until either it acquires the lock (a surviving session becomes the sole
// writer once the holder exits) or ctx is cancelled.
//
// On ErrLockLive, onDeferred is invoked once per retry (the caller logs the
// "deferring to it" line and may no-op subsequent calls) and the loop then
// sleeps jitter(interval), honoring ctx.Done() so a cancellation during the
// sleep returns ctx.Err() promptly rather than waiting out the interval.
// Any other outcome — nil (clean shutdown), watch.ErrWatchDisabled, or a
// genuine non-ErrLockLive error — returns immediately without ever calling
// onDeferred: policy is terminal (it doesn't change mid-session) and a
// genuine error is not something retrying can fix.
//
// D-16 confirms acquire() (lock.go) self-heals a stale lock on every
// independent call — a crashed holder is detected and cleared the very
// next retry's acquire(), not by any new liveness/staleness machinery
// added here. This loop is therefore nothing more than "call Run again":
// no poll, no wait-for-pid, no watch-for-exit.
func RunWithRetry(ctx context.Context, d *Daemon, interval time.Duration, onDeferred func()) error {
	for {
		err := d.Run(ctx)
		if err == nil || !errors.Is(err, ErrLockLive) {
			return err
		}
		onDeferred()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter(interval)):
		}
	}
}

// jitter returns interval plus a bounded pseudo-random positive amount (up
// to ~20% of interval) so concurrent RunWithRetry loops retrying against
// the same contended lockfile do not thunder-herd on a synchronized
// cadence. Not crypto-grade — math/rand is sufficient for a scheduling
// stagger. Always >= interval (never negative, never less than interval).
func jitter(interval time.Duration) time.Duration {
	spread := interval / 5
	if spread <= 0 {
		return interval
	}
	return interval + time.Duration(rand.Int63n(int64(spread)))
}

// flush is the debouncer's callback (RESEARCH Pattern 3/7): it runs in the
// debounce timer's own goroutine. indexer.Sync itself re-diffs the whole
// repo (stat pre-filter -> content-hash confirm, D-01a) rather than being
// scoped to the specific changed paths the debouncer coalesced, so the
// path set argument carries no information Sync needs — it exists purely
// to prove to the caller that a change was observed. The sidecar is
// touched BEFORE syncing (so a concurrent Status()/Explore() read reports
// stale for the whole sync duration, not just the debounce window) and
// cleared only after a successful commit; a failed sync leaves the
// sidecar in place so the next successful sync (or the no-daemon mtime
// fallback, D-04a) is the only thing that clears staleness.
//
// flush returns Sync's error so Run's requeue wrapper (CR-01) can branch on
// graphstore.ErrStoreLocked (via errors.Is — the sentinel is only ever
// attached inside graphstore.Open, so a non-lock filesystem error in Sync's
// chain can never trigger a requeue); the error is already logged here, so
// the wrapper never double-reports it.
func (d *Daemon) flush(_ map[string]struct{}) error {
	if d.onSyncStart != nil {
		d.onSyncStart()
	}

	if err := d.touchPending(); err != nil {
		log.Printf("daemon: touching pending sidecar: %v", err)
	}

	syncFn := indexer.Sync
	if d.syncFn != nil {
		syncFn = d.syncFn
	}
	d.syncMu.Lock()
	stats, err := syncFn(d.repoRoot, d.storeDir, d.opts)
	d.syncMu.Unlock()

	if err != nil {
		log.Printf("daemon: sync: %v", err)
	} else if clearErr := d.clearPending(); clearErr != nil {
		log.Printf("daemon: clearing pending sidecar: %v", clearErr)
	}

	if d.onSync != nil {
		d.onSync(stats, err)
	}
	return err
}

func (d *Daemon) touchPending() error {
	return os.WriteFile(filepath.Join(d.codegraphDir, staleSidecarName), nil, 0o644)
}

func (d *Daemon) clearPending() error {
	err := os.Remove(filepath.Join(d.codegraphDir, staleSidecarName))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
