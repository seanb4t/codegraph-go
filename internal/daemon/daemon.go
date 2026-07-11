package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

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
}

// New resolves repoRoot's .codegraph/ layout and returns a Daemon ready to
// Run. It does not touch the lockfile or open the store — those happen
// inside Run, so a not-yet-started Daemon never holds any process-wide
// resource. New fails with ErrNotInitialized if repoRoot has no
// .codegraph/ (mirrors internal/cli's index/sync guidance: run `codegraph
// init` first).
func New(repoRoot string) (*Daemon, error) {
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
	return &Daemon{
		repoRoot:     abs,
		codegraphDir: codegraphDir,
		storeDir:     filepath.Join(codegraphDir, storeDirName),
	}, nil
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
// immediately without starting a watcher.
func (d *Daemon) Run(ctx context.Context) error {
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

	deb := watch.NewDebouncer(ctx, watch.DebounceDuration(), d.flush)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.Run(ctx, deb)
	}()

	<-ctx.Done()
	wg.Wait()
	// CR-01: wg.Wait() only joins the tracked watcher goroutine — it does
	// NOT join a debounce flush that had already started running its
	// indexer.Sync (on the timer's own untracked goroutine) when ctx was
	// cancelled. watchLoop's deb.Stop() (called above, inside w.Run) can
	// only cancel a timer that hasn't fired yet; deb.Wait() is the
	// explicit join for one that has, closing the window where Run
	// released the daemon lock (see the deferred release() above) while a
	// Sync was still mid-commit against the single coordinated Writer
	// (INDX-05, D-07, SYNC-06).
	deb.Wait()
	return nil
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
func (d *Daemon) flush(_ map[string]struct{}) {
	if d.onSyncStart != nil {
		d.onSyncStart()
	}

	if err := d.touchPending(); err != nil {
		log.Printf("daemon: touching pending sidecar: %v", err)
	}

	d.syncMu.Lock()
	stats, err := indexer.Sync(d.repoRoot, d.storeDir, d.opts)
	d.syncMu.Unlock()

	if err != nil {
		log.Printf("daemon: sync: %v", err)
	} else if clearErr := d.clearPending(); clearErr != nil {
		log.Printf("daemon: clearing pending sidecar: %v", clearErr)
	}

	if d.onSync != nil {
		d.onSync(stats, err)
	}
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
