package graphstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/graphstore/locktest"
	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// commitNode commits one node with the given id through s.
func commitNode(t fataler, s GraphStore, id string) {
	t.Helper()
	w, err := s.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutNode(&schema.Node{Id: id, Name: id}); err != nil {
		t.Fatalf("PutNode(%s): %v", id, err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// nodeVisible reports whether a new snapshot of s holds the node id.
func nodeVisible(t fataler, s GraphStore, id string) bool {
	t.Helper()
	r, err := s.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	_, err = r.GetNode(id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetNode(%s): %v", id, err)
	}
	return err == nil
}

// mustOpen opens the store at dir.
func mustOpen(t fataler, dir string) *pebbleStore {
	t.Helper()
	s, err := open(dir, false)
	if err != nil {
		t.Fatalf("Open(%s): %v", dir, err)
	}
	return s
}

// mustClose closes s.
func mustClose(t fataler, s GraphStore) {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// sharedHandleCount returns the number of handles in the table.
func sharedHandleCount() int {
	sharedHandles.mu.Lock()
	defer sharedHandles.mu.Unlock()
	return len(sharedHandles.entries)
}

// requireNoSharedHandle fails the test if the table still holds a handle.
func requireNoSharedHandle(t fataler) {
	t.Helper()
	if n := sharedHandleCount(); n != 0 {
		t.Fatalf("the table holds %d handles, want none", n)
	}
}

// setSharedOpenHook sets sharedOpenHook for the test.
func setSharedOpenHook(t *testing.T, hook func()) {
	t.Helper()
	sharedOpenHook = hook
	t.Cleanup(func() { sharedOpenHook = nil })
}

// setSharedWaitHook sets sharedWaitHook for the test.
func setSharedWaitHook(t *testing.T, hook func(woke bool)) {
	t.Helper()
	sharedWaitHook = hook
	t.Cleanup(func() { sharedWaitHook = nil })
}

// setSharedCloseHook sets sharedCloseHook for the test.
func setSharedCloseHook(t *testing.T, hook func()) {
	t.Helper()
	sharedCloseHook = hook
	t.Cleanup(func() { sharedCloseHook = nil })
}

func TestOpenSecondOpenInProcessSharesHandle(t *testing.T) {
	sleeps := countRetrySleeps(t)
	dir := t.TempDir()

	first := mustOpen(t, dir)
	second := mustOpen(t, dir)
	if first.handle != second.handle {
		t.Fatalf("the second Open did not take the handle of the first")
	}
	if n := sleeps.Load(); n != 0 {
		t.Fatalf("the second Open slept %d times, want 0", n)
	}

	commitNode(t, first, "written-through-first")
	if !nodeVisible(t, second, "written-through-first") {
		t.Fatalf("a snapshot of the second handle does not hold a record written through the first")
	}
	mustClose(t, first)
	mustClose(t, second)
	requireNoSharedHandle(t)
}

func TestOpenSpellingsShareHandle(t *testing.T) {
	noRetrySleep(t)
	base := realpathtest.TempDir(t)
	t.Chdir(base)
	if err := os.Mkdir("a", 0o755); err != nil {
		t.Fatal(err)
	}
	// A link to the parent directory gives one more name for each store
	// (#286).
	if err := os.Symlink(base, "lnk"); err != nil {
		t.Fatal(err)
	}

	stores := 0
	rapid.Check(t, func(rt *rapid.T) {
		// Each run uses a store that does not exist before its first Open.
		stores++
		name := fmt.Sprintf("store%d", stores)
		spellings := []string{
			name,
			"./" + name,
			"a/../" + name,
			name + "/",
			".//" + name,
			filepath.Join(base, name),
			filepath.Join(base, name) + "/",
			"lnk/" + name,
			filepath.Join(base, "lnk", name),
		}
		drawn := rapid.SliceOfN(rapid.SampledFrom(spellings), 2, 6).Draw(rt, "spellings")

		handles := make([]*pebbleStore, len(drawn))
		for i, spelling := range drawn {
			s, err := open(spelling, false)
			if err != nil {
				rt.Fatalf("Open(%q): %v", spelling, err)
			}
			handles[i] = s
			if s.handle != handles[0].handle {
				rt.Fatalf("Open(%q) did not take the handle of Open(%q)", spelling, drawn[0])
			}
			commitNode(rt, s, fmt.Sprintf("n%d", i))
		}
		for i, s := range handles {
			for j := range handles {
				if !nodeVisible(rt, s, fmt.Sprintf("n%d", j)) {
					rt.Fatalf("the handle of %q does not hold the record written through %q", drawn[i], drawn[j])
				}
			}
		}
		for _, s := range handles {
			mustClose(rt, s)
		}
		requireNoSharedHandle(rt)
		// The Pebble lock is free under every drawn spelling.
		for _, spelling := range drawn {
			locktest.HoldStoreLock(t, spelling)()
		}
	})
}

func TestSharedHandleClose(t *testing.T) {
	dir := t.TempDir()
	first := mustOpen(t, dir)
	second := mustOpen(t, dir)
	commitNode(t, first, "n")

	mustClose(t, first)
	if _, err := first.Snapshot(); !errors.Is(err, ErrClosed) {
		t.Fatalf("Snapshot on the closed handle: %v, want ErrClosed", err)
	}
	if !nodeVisible(t, second, "n") {
		t.Fatalf("the open handle lost its record when the other handle closed")
	}

	// A second Close of one handle does not take a user from the other.
	if err := first.Close(); err != nil {
		t.Fatalf("a second Close: %v, want nil", err)
	}
	if !nodeVisible(t, second, "n") {
		t.Fatalf("the open handle stopped reading after a second Close of the other handle")
	}
	if n := sharedHandleCount(); n != 1 {
		t.Fatalf("the table holds %d handles with one user open, want 1", n)
	}

	mustClose(t, second)
	requireNoSharedHandle(t)
	locktest.HoldStoreLock(t, dir)()
}

func TestOpenDuringLastCloseRetries(t *testing.T) {
	// holdLastClose opens the store at dir, starts its last Close and
	// returns when that Close is in the closing state. unblock lets the
	// Close go on and waits for it.
	holdLastClose := func(t *testing.T, dir string) (unblock func()) {
		t.Helper()
		s := mustOpen(t, dir)
		inClose, goOn, closed := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		setSharedCloseHook(t, func() {
			close(inClose)
			<-goOn
		})
		go func() { closed <- s.Close() }()
		<-inClose
		return sync.OnceFunc(func() {
			sharedCloseHook = nil
			close(goOn)
			if err := <-closed; err != nil {
				t.Errorf("the held Close: %v", err)
			}
		})
	}

	t.Run("the close ends during the retry", func(t *testing.T) {
		dir := t.TempDir()
		unblock := holdLastClose(t, dir)
		sleeps := 0
		setRetrySleep(t, func() {
			sleeps++
			unblock()
		})

		s := mustOpen(t, dir)
		if sleeps != 1 {
			t.Fatalf("Open slept %d times, want 1: one attempt met the closing handle", sleeps)
		}
		mustClose(t, s)
	})

	t.Run("the close lasts through every attempt", func(t *testing.T) {
		dir := t.TempDir()
		unblock := holdLastClose(t, dir)
		defer unblock()
		sleeps := countRetrySleeps(t)

		if _, err := Open(dir); !errors.Is(err, ErrStoreLocked) {
			t.Fatalf("Open during a close that does not end: %v, want ErrStoreLocked", err)
		}
		if n := sleeps.Load(); n != openLockRetryAttempts-1 {
			t.Fatalf("Open slept %d times, want %d", n, openLockRetryAttempts-1)
		}
	})

	t.Run("the close returns an error", func(t *testing.T) {
		dir := t.TempDir()
		s := mustOpen(t, dir)
		// Pebble's Close reports an iterator that is still open.
		iter, err := s.db.NewIter(nil)
		if err != nil {
			t.Fatalf("NewIter: %v", err)
		}
		if err := s.Close(); err == nil {
			t.Fatalf("Close with an open iterator returned no error")
		}
		_ = iter.Close()
		requireNoSharedHandle(t)

		sleeps := countRetrySleeps(t)
		mustClose(t, mustOpen(t, dir))
		if n := sleeps.Load(); n != 0 {
			t.Fatalf("Open after a failed close slept %d times, want 0", n)
		}
	})

	t.Run("one budget covers the close and a held Pebble lock", func(t *testing.T) {
		dir := t.TempDir()
		unblock := holdLastClose(t, dir)
		sleeps := 0
		setRetrySleep(t, func() {
			sleeps++
			if sleeps == 2 {
				unblock()
				locktest.HoldStoreLock(t, dir)
			}
		})

		if _, err := Open(dir); !errors.Is(err, ErrStoreLocked) {
			t.Fatalf("Open: %v, want ErrStoreLocked", err)
		}
		if sleeps != openLockRetryAttempts-1 {
			t.Fatalf("Open slept %d times, want %d in total for both locks", sleeps, openLockRetryAttempts-1)
		}
	})
}

// TestSharedHandleConcurrentOpenClose checks the count of users against a
// model, for any order of opens and closes of one store by several users.
func TestSharedHandleConcurrentOpenClose(t *testing.T) {
	sleeps := countRetrySleeps(t)
	dir := t.TempDir()

	rapid.Check(t, func(rt *rapid.T) {
		var users []*pebbleStore
		steps := rapid.SliceOfN(rapid.Bool(), 1, 40).Draw(rt, "steps")
		for _, opens := range steps {
			if opens || len(users) == 0 {
				s, err := open(dir, false)
				if err != nil {
					rt.Fatalf("Open with %d users: %v", len(users), err)
				}
				users = append(users, s)
			} else {
				i := rapid.IntRange(0, len(users)-1).Draw(rt, "user")
				mustClose(rt, users[i])
				users = slices.Delete(users, i, i+1)
			}
			if n, want := sharedHandleCount(), min(len(users), 1); n != want {
				rt.Fatalf("the table holds %d handles with %d users, want %d", n, len(users), want)
			}
			// Every user that is still open reads through the handle.
			for _, s := range users {
				nodeVisible(rt, s, "any")
			}
		}
		for _, s := range users {
			mustClose(rt, s)
		}
		requireNoSharedHandle(rt)
		locktest.HoldStoreLock(t, dir)()
	})
	if n := sleeps.Load(); n != 0 {
		t.Fatalf("the opens slept %d times, want 0", n)
	}
}

func TestSharedHandleWaitersShareOpen(t *testing.T) {
	// blockFirstOpen makes the first open of a shared handle wait in the
	// opening state. It returns a channel that closes when that open is
	// there, and a function that lets it go on.
	blockFirstOpen := func(t *testing.T) (inOpen <-chan struct{}, unblock func()) {
		t.Helper()
		in, goOn := make(chan struct{}), make(chan struct{})
		var calls atomic.Int64
		setSharedOpenHook(t, func() {
			if calls.Add(1) == 1 {
				close(in)
				<-goOn
			}
		})
		return in, sync.OnceFunc(func() { close(goOn) })
	}
	type result struct {
		s   *pebbleStore
		err error
	}
	// openAsync starts an open of dir and returns the channel of its result.
	openAsync := func(dir string) <-chan result {
		out := make(chan result, 1)
		go func() {
			s, err := open(dir, false)
			out <- result{s, err}
		}()
		return out
	}

	t.Run("the open succeeds", func(t *testing.T) {
		sleeps := countRetrySleeps(t)
		dir := t.TempDir()
		inOpen, unblock := blockFirstOpen(t)

		waiting := make(chan struct{})
		signalWaiting := sync.OnceFunc(func() { close(waiting) })
		setSharedWaitHook(t, func(woke bool) {
			// A caller can wait again on its next attempt.
			if !woke {
				signalWaiting()
			}
		})

		first := openAsync(dir)
		<-inOpen
		second := openAsync(dir)
		// The second Open found the handle in the opening state and waits.
		<-waiting
		unblock()

		a, b := <-first, <-second
		if a.err != nil || b.err != nil {
			t.Fatalf("Open = %v and %v, want both to succeed", a.err, b.err)
		}
		if a.s.handle != b.s.handle {
			t.Fatalf("the waiter did not take the handle of the opener")
		}
		if n := sleeps.Load(); n != 0 {
			t.Fatalf("the opens slept %d times, want 0", n)
		}

		commitNode(t, a.s, "n")
		mustClose(t, a.s)
		if !nodeVisible(t, b.s, "n") {
			t.Fatalf("the waiter stopped reading when the opener closed")
		}
		if n := sharedHandleCount(); n != 1 {
			t.Fatalf("the table holds %d handles with the waiter open, want 1", n)
		}
		mustClose(t, b.s)
		requireNoSharedHandle(t)
		locktest.HoldStoreLock(t, dir)()
	})

	t.Run("the open fails with an error that is not a held lock", func(t *testing.T) {
		sleeps := countRetrySleeps(t)
		dir := filepath.Join(t.TempDir(), "store")
		if err := os.WriteFile(dir, []byte("not a store"), 0o644); err != nil {
			t.Fatal(err)
		}
		inOpen, unblock := blockFirstOpen(t)

		first := openAsync(dir)
		<-inOpen
		second := openAsync(dir)
		unblock()

		a, b := <-first, <-second
		if a.err == nil || errors.Is(a.err, ErrStoreLocked) {
			t.Fatalf("Open of a regular file: %v, want an error that is not ErrStoreLocked", a.err)
		}
		if b.err == nil || errors.Is(b.err, ErrStoreLocked) {
			t.Fatalf("the second Open of a regular file: %v, want an error that is not ErrStoreLocked", b.err)
		}
		if n := sleeps.Load(); n != 0 {
			t.Fatalf("the opens slept %d times, want 0", n)
		}
		requireNoSharedHandle(t)

		if err := os.Remove(dir); err != nil {
			t.Fatal(err)
		}
		mustClose(t, mustOpen(t, dir))
	})

	t.Run("the open fails on a held lock", func(t *testing.T) {
		dir := t.TempDir()
		release := locktest.HoldStoreLock(t, dir)
		inOpen, unblock := blockFirstOpen(t)
		waiting := make(chan struct{})
		signalWaiting := sync.OnceFunc(func() { close(waiting) })
		setSharedWaitHook(t, func(woke bool) {
			// A caller can wait again on its next attempt.
			if !woke {
				signalWaiting()
			}
		})
		// The opener and the waiter each get the lock error of the one
		// shared attempt, and each sleeps one time. The first of those
		// sleeps frees the lock, so the next attempt of each succeeds.
		var sleeps atomic.Int64
		setRetrySleep(t, func() {
			if sleeps.Add(1) == 1 {
				release()
			}
		})

		first := openAsync(dir)
		<-inOpen
		second := openAsync(dir)
		<-waiting
		unblock()

		a, b := <-first, <-second
		if a.err != nil || b.err != nil {
			t.Fatalf("Open = %v and %v, want both to succeed after the release", a.err, b.err)
		}
		if a.s.handle != b.s.handle {
			t.Fatalf("the two opens did not end on one handle")
		}
		if n := sleeps.Load(); n != 2 {
			t.Fatalf("the opens slept %d times, want 2: one for each caller of the failed attempt", n)
		}
		mustClose(t, a.s)
		mustClose(t, b.s)
		requireNoSharedHandle(t)
	})

	t.Run("the opener closes before the waiter looks again", func(t *testing.T) {
		sleeps := countRetrySleeps(t)
		dir := t.TempDir()
		inOpen, unblock := blockFirstOpen(t)
		waiting, openerClosed := make(chan struct{}), make(chan struct{})
		setSharedWaitHook(t, func(woke bool) {
			if !woke {
				close(waiting)
				return
			}
			<-openerClosed
		})

		first := openAsync(dir)
		<-inOpen
		second := openAsync(dir)
		<-waiting
		unblock()
		a := <-first
		if a.err != nil {
			t.Fatalf("the first Open: %v", a.err)
		}
		openerHandle := a.s.handle
		mustClose(t, a.s)
		close(openerClosed)

		// The waiter finds the handle of the opener closed. It must not
		// take that handle: it retries one time and opens a new one.
		b := <-second
		if b.err != nil {
			t.Fatalf("the second Open: %v", b.err)
		}
		if b.s.handle == openerHandle {
			t.Fatalf("the waiter took the handle that its opener already closed")
		}
		if n := sleeps.Load(); n != 1 {
			t.Fatalf("the waiter slept %d times, want 1", n)
		}
		commitNode(t, b.s, "n")
		if !nodeVisible(t, b.s, "n") {
			t.Fatalf("the waiter does not read through its new handle")
		}
		mustClose(t, b.s)
		requireNoSharedHandle(t)
	})
}

func TestSharedHandleWaiterKeepsOwnBudget(t *testing.T) {
	t.Run("a later open outlasts an earlier one", func(t *testing.T) {
		dir := t.TempDir()
		release := locktest.HoldStoreLock(t, dir)

		// The first Open sleeps openLockRetryAttempts-1 times. The second
		// Open starts in the last of those sleeps. Each later sleep is one
		// of the second Open, and it waits for the test.
		var (
			sleeps    atomic.Int64
			secondErr = make(chan error, 1)
			goOn      = make(chan struct{})
		)
		setRetrySleep(t, func() {
			switch n := sleeps.Add(1); {
			case n == openLockRetryAttempts-1:
				go func() {
					s, err := open(dir, false)
					if err == nil {
						err = s.Close()
					}
					secondErr <- err
				}()
			case n > openLockRetryAttempts-1:
				<-goOn
			}
		})

		if _, err := Open(dir); !errors.Is(err, ErrStoreLocked) {
			t.Fatalf("the first Open: %v, want ErrStoreLocked", err)
		}
		release()
		close(goOn)
		if err := <-secondErr; err != nil {
			t.Fatalf("the second Open, with attempts of its own left: %v", err)
		}
		requireNoSharedHandle(t)
	})

	t.Run("each open has its own attempts", func(t *testing.T) {
		sleeps := countRetrySleeps(t)
		dir := t.TempDir()
		locktest.HoldStoreLock(t, dir)

		const opens = 3
		var wg sync.WaitGroup
		errs := make([]error, opens)
		for i := range opens {
			wg.Go(func() { _, errs[i] = Open(dir) })
		}
		wg.Wait()
		for i, err := range errs {
			if !errors.Is(err, ErrStoreLocked) {
				t.Errorf("Open %d: %v, want ErrStoreLocked", i, err)
			}
		}
		if n, want := sleeps.Load(), int64(opens*(openLockRetryAttempts-1)); n != want {
			t.Fatalf("the opens slept %d times, want %d: %d for each open", n, want, openLockRetryAttempts-1)
		}
		requireNoSharedHandle(t)
	})
}

func TestOpenOfSecondPathDoesNotWait(t *testing.T) {
	sleeps := countRetrySleeps(t)
	dirA, dirB := t.TempDir(), t.TempDir()

	inOpen, goOn := make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	setSharedOpenHook(t, func() {
		if calls.Add(1) == 1 {
			close(inOpen)
			<-goOn
		}
	})
	first := make(chan error, 1)
	go func() {
		s, err := open(dirA, false)
		if err == nil {
			err = s.Close()
		}
		first <- err
	}()
	<-inOpen

	// Open(dirB) returns while Open(dirA) is still in the opening state.
	mustClose(t, mustOpen(t, dirB))
	select {
	case err := <-first:
		t.Fatalf("Open(dirA) returned (%v) before the test let it go on", err)
	default:
	}
	if n := sleeps.Load(); n != 0 {
		t.Fatalf("Open(dirB) slept %d times, want 0", n)
	}

	close(goOn)
	if err := <-first; err != nil {
		t.Fatalf("Open(dirA): %v", err)
	}
	requireNoSharedHandle(t)
}

func TestReplaceStoreWaitsForSharedHandle(t *testing.T) {
	noRetrySleep(t)
	staging, live := storePaths(t)
	oldExport := buildStore(t, live, "old")
	newExport := buildStore(t, staging, "new")
	first := mustOpen(t, live)
	second := mustOpen(t, live)

	mustClose(t, first)
	if err := ReplaceStore(staging, live); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("ReplaceStore with one of two handles closed: %v, want ErrStoreLocked", err)
	}
	if !bytes.Equal(exportOf(t, second), oldExport) {
		t.Fatalf("the live store changed in a ReplaceStore that failed")
	}

	mustClose(t, second)
	if err := ReplaceStore(staging, live); err != nil {
		t.Fatalf("ReplaceStore with both handles closed: %v", err)
	}
	if !bytes.Equal(exportAt(t, live), newExport) {
		t.Errorf("the live path does not hold the new store")
	}
}

// TestStagingStoreHasOneOwner checks that a staging store never shares its
// handle: a second open of its directory in this process is refused.
func TestStagingStoreHasOneOwner(t *testing.T) {
	sleeps := countRetrySleeps(t)
	dir := t.TempDir()
	staging, err := OpenStaging(dir)
	if err != nil {
		t.Fatalf("OpenStaging: %v", err)
	}

	if _, err := OpenStaging(dir); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("a second OpenStaging: %v, want ErrStoreLocked", err)
	}
	if _, err := Open(dir); !errors.Is(err, ErrStoreLocked) {
		t.Fatalf("Open of a staging store that is open: %v, want ErrStoreLocked", err)
	}
	if n, want := sleeps.Load(), int64(2*(openLockRetryAttempts-1)); n != want {
		t.Fatalf("the two refused opens slept %d times, want %d", n, want)
	}
	requireNoSharedHandle(t)

	// Positive assertion: the directory opens after the staging store closes.
	mustClose(t, staging)
	mustClose(t, mustOpen(t, dir))
}

// TestSharedHandleParallelOpenClose runs drawn scripts of opens and closes
// of one store on several goroutines. An open that meets the last close of
// another goroutine can use up its retry, so the property allows
// ErrStoreLocked. Every open that succeeds reads, and the process holds no
// handle after the last close.
func TestSharedHandleParallelOpenClose(t *testing.T) {
	noRetrySleep(t)
	dir := t.TempDir()
	readThenClose := func(s *pebbleStore) error {
		r, err := s.Snapshot()
		if err != nil {
			return errors.Join(err, s.Close())
		}
		_, err = r.GetMeta()
		if errors.Is(err, ErrNotFound) {
			err = nil
		}
		return errors.Join(err, r.Close(), s.Close())
	}

	rapid.Check(t, func(rt *rapid.T) {
		scripts := rapid.SliceOfN(rapid.IntRange(1, 6), 2, 6).Draw(rt, "cycles of each goroutine")
		var (
			wg     sync.WaitGroup
			opened atomic.Int64
			errs   = make([]error, len(scripts))
		)
		for g, cycles := range scripts {
			wg.Go(func() {
				for range cycles {
					s, err := open(dir, false)
					if errors.Is(err, ErrStoreLocked) {
						continue
					}
					if err != nil {
						errs[g] = err
						return
					}
					opened.Add(1)
					if err := readThenClose(s); err != nil {
						errs[g] = err
						return
					}
				}
			})
		}
		wg.Wait()
		for g, err := range errs {
			if err != nil {
				rt.Fatalf("goroutine %d: %v", g, err)
			}
		}
		// Positive assertion: the first open of a run meets no handle, so
		// some open of every run succeeds.
		if opened.Load() == 0 {
			rt.Fatalf("no open succeeded")
		}
		requireNoSharedHandle(rt)
		locktest.HoldStoreLock(t, dir)()
	})
}
