package daemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// TestRunChecksIndexBeforeLock covers the start-time check of #375: acquire
// writes through .codegraph, so a link that leads outside must stop Run
// before acquire on the first call and on every retry of RunWithRetry. The
// stale lock in outside is what acquire would replace.
func TestRunChecksIndexBeforeLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		// before runs after New and before RunWithRetry.
		before func(t *testing.T, root, outside string)
		// deferred is the onDeferred callback, and wantDeferred the number
		// of times RunWithRetry calls it.
		deferred     func(t *testing.T, root, outside string)
		wantDeferred int
	}{
		{
			name: "first call",
			before: func(t *testing.T, root, outside string) {
				replaceWithLink(t, filepath.Join(root, codegraphDirName), outside)
			},
			deferred:     func(*testing.T, string, string) {},
			wantDeferred: 0,
		},
		{
			name: "retry",
			before: func(t *testing.T, root, _ string) {
				writeLockInfo(t, filepath.Join(root, codegraphDirName), lockInfo{PID: os.Getpid(), StartedAt: selfStartedAt()})
			},
			deferred: func(t *testing.T, root, outside string) {
				replaceWithLink(t, filepath.Join(root, codegraphDirName), outside)
			},
			wantDeferred: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := mkIndexDirs(t)
			outside := t.TempDir()
			writeLock(t, outside, deadPID(t))
			lock, err := os.ReadFile(lockPath(outside))
			if err != nil {
				t.Fatal(err)
			}
			d, err := New(root, indexer.Options{Quiet: true}, WithProbe(watch.Probe{IsWSL: func() bool { return false }}))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			tc.before(t, root, outside)

			deferred := 0
			err = RunWithRetry(t.Context(), d, time.Millisecond, func() {
				deferred++
				tc.deferred(t, root, outside)
			})
			if !errors.Is(err, query.ErrIndexOutsideRoot) {
				t.Fatalf("RunWithRetry = %v, want query.ErrIndexOutsideRoot", err)
			}
			if deferred != tc.wantDeferred {
				t.Fatalf("onDeferred ran %d times, want %d", deferred, tc.wantDeferred)
			}
			if got, err := os.ReadFile(lockPath(outside)); err != nil || !bytes.Equal(got, lock) {
				t.Fatalf("the lock outside changed: %q (err %v), want %q", got, err, lock)
			}
		})
	}
}

// TestRunReleasesOnlyItsOwnLock covers #375's shutdown: Run removes
// daemon.lock only where .codegraph now resolves inside the repository and
// only when the lock names this process. Each row changes the index while
// Run holds its lock, then cancels Run.
func TestRunReleasesOnlyItsOwnLock(t *testing.T) {
	for _, tc := range []struct {
		name string
		// apply changes the index and returns the directory whose
		// daemon.lock the row checks.
		apply func(t *testing.T, root string) string
		keep  bool
	}{
		{
			// The other lock names this pid, so only the path check keeps it.
			name: "index linked to another project",
			apply: func(t *testing.T, root string) string {
				other := filepath.Join(t.TempDir(), codegraphDirName)
				if err := os.Mkdir(other, 0o755); err != nil {
					t.Fatal(err)
				}
				writeLockInfo(t, other, lockInfo{PID: os.Getpid(), StartedAt: selfStartedAt()})
				replaceWithLink(t, filepath.Join(root, codegraphDirName), other)
				return other
			},
			keep: true,
		},
		{
			// The path is unchanged, so only the owner check keeps it.
			name: "lock taken by another process",
			apply: func(t *testing.T, root string) string {
				dir := filepath.Join(root, codegraphDirName)
				writeLock(t, dir, os.Getppid())
				return dir
			},
			keep: true,
		},
		{
			name: storeLinkedOutside.name,
			apply: func(t *testing.T, root string) string {
				storeLinkedOutside.apply(t, root)
				return filepath.Join(root, codegraphDirName)
			},
		},
		{name: indexMovedInside.name, apply: indexMovedInside.apply},
		{
			name: indexUnchanged.name,
			apply: func(t *testing.T, root string) string {
				return filepath.Join(root, codegraphDirName)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := mkIndexDirs(t)
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				r := startCaughtUpFakeRun(t, root, ctx, cancel, nil)

				lockDir := tc.apply(t, root)
				before, _ := os.ReadFile(lockPath(lockDir))
				cancel()
				if err := receive(t, r.runErr, "Run to return after cancel"); err != nil {
					t.Fatalf("Run: %v", err)
				}

				after, err := os.ReadFile(lockPath(lockDir))
				switch {
				case tc.keep && (err != nil || !bytes.Equal(after, before)):
					t.Fatalf("%s changed: %q (err %v), want %q", lockPath(lockDir), after, err, before)
				case !tc.keep && !os.IsNotExist(err):
					t.Fatalf("%s is still there (err %v), want it removed", lockPath(lockDir), err)
				}
			})
		})
	}
}

// pathsError is an error value that == cannot compare: comparing two of
// them panics at run time.
type pathsError struct{ paths []string }

func (e pathsError) Error() string { return fmt.Sprintf("stopped for %v", e.paths) }

// TestRunReturnsNilWhenParentEnds covers the clean-stop rule of design D3:
// whatever cause ends the parent context, Run returns nil, because a stop
// that the caller asked for is not an error.
func TestRunReturnsNilWhenParentEnds(t *testing.T) {
	for _, tc := range []struct {
		name string
		// start returns the parent context, the function that ends it, and
		// a cancel for the test's join.
		start func() (context.Context, func(), context.CancelFunc)
	}{
		{"cancel", func() (context.Context, func(), context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			return ctx, cancel, cancel
		}},
		{"deadline", func() (context.Context, func(), context.CancelFunc) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			return ctx, func() { time.Sleep(2 * time.Hour) }, cancel
		}},
		{"cause of the caller", func() (context.Context, func(), context.CancelFunc) {
			ctx, cancel := context.WithCancelCause(context.Background())
			return ctx, func() { cancel(errors.New("the caller stopped")) }, func() { cancel(nil) }
		}},
		{"cause that == cannot compare", func() (context.Context, func(), context.CancelFunc) {
			ctx, cancel := context.WithCancelCause(context.Background())
			return ctx, func() { cancel(pathsError{paths: []string{"a"}}) }, func() { cancel(nil) }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := mkIndexDirs(t)
			synctest.Test(t, func(t *testing.T) {
				ctx, end, cancel := tc.start()
				r := startCaughtUpFakeRun(t, root, ctx, cancel, nil)
				end()
				if err := receive(t, r.runErr, "Run to return after its parent ended"); err != nil {
					t.Fatalf("Run = %v, want nil", err)
				}
			})
		})
	}
}

// TestRunStopsWhenIndexChanges covers the flush-time check of #375: after
// a catch-up flush with the index unchanged, a pull changes the index, and
// the next event stops Run with the error of checkIndex and no second sync.
func TestRunStopsWhenIndexChanges(t *testing.T) {
	for _, tc := range []struct {
		change indexChange
		// empty is true when the target held nothing before the change, so
		// any entry in it after Run is a write through the changed link.
		empty bool
		check func(t *testing.T, err error)
	}{
		{indexLinkedOutside, true, wantOutside},
		{storeLinkedOutside, true, wantOutside},
		{indexLinkedToItself, false, func(t *testing.T, err error) {
			if err == nil || errors.Is(err, ErrIndexChanged) || errors.Is(err, ErrWatcherClosed) {
				t.Fatalf("Run = %v, want the resolve error of checkIndex", err)
			}
		}},
		{indexMovedInside, false, func(t *testing.T, err error) {
			if changed := wantChanged(t, err); changed.Was.Dir == changed.Now.Dir || changed.Was.Store == changed.Now.Store {
				t.Fatalf("Dir and Store did not both change: %+v", changed)
			}
		}},
		{storeMovedInside, true, func(t *testing.T, err error) {
			changed := wantChanged(t, err)
			for _, p := range []string{changed.Was.Store, changed.Now.Store} {
				if !strings.Contains(err.Error(), p) {
					t.Errorf("message %q does not contain %q", err.Error(), p)
				}
			}
		}},
	} {
		t.Run(tc.change.name, func(t *testing.T) {
			root := mkIndexDirs(t)
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				r := startCaughtUpFakeRun(t, root, ctx, cancel, nil)

				target := tc.change.apply(t, root)
				r.fw.events <- "main.go"
				tc.check(t, receive(t, r.runErr, "Run to stop after the index changed"))

				if got := r.syncs.Load(); got != 1 {
					t.Fatalf("syncFn ran %d times, want 1: only the catch-up sync", got)
				}
				if target == "" {
					return
				}
				entries, err := os.ReadDir(target)
				if err != nil {
					t.Fatal(err)
				}
				for _, e := range entries {
					if tc.empty || e.Name() == staleSidecarName {
						t.Errorf("Run wrote %s through the changed link", filepath.Join(target, e.Name()))
					}
				}
			})
		})
	}
}
