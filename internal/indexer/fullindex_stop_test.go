package indexer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"go.uber.org/goleak"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// stepCounts counts the claims and the releases of each step of a run.
type stepCounts struct {
	mu       sync.Mutex
	claims   map[string]int
	releases map[string]int
}

func countSteps(t *testing.T) *stepCounts {
	t.Helper()
	c := &stepCounts{claims: map[string]int{}, releases: map[string]int{}}
	setHook(t, &onClaim, func(step string, _ int) error {
		c.mu.Lock()
		c.claims[step]++
		c.mu.Unlock()
		return nil
	})
	setHook(t, &onRelease, func(step string, _ int, _ *runState) {
		c.mu.Lock()
		c.releases[step]++
		c.mu.Unlock()
	})
	return c
}

func (c *stepCounts) of(step string) (claims, releases int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.claims[step], c.releases[step]
}

// scratchDirs lists the scratch directories that a run left beside the
// store at storeDir.
func scratchDirs(t *testing.T, storeDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(storeDir))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(storeDir)+".scratch") {
			out = append(out, e.Name())
		}
	}
	return out
}

// stalledRun starts a full index whose writer stops inside its first unit:
// the batch cap is 0, so the writer commits before the second record of
// that unit, and onCommit blocks there until the test lets it go or the
// context ends. It returns the channel of the result of the run, the state
// of pass 3, and the function that lets the writer go.
func stalledRun(t *testing.T, ctx context.Context, workers int) (done chan error, state func() *runState, letGo func()) {
	t.Helper()
	root := goTree(t, 500, false)
	storeDir := filepath.Join(t.TempDir(), "store")
	setHook(t, &batchCap, 0)
	gate := make(chan struct{})
	var once sync.Once
	setHook(t, &onCommit, func(graphstore.Commit) error {
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	var st *runState
	setHook(t, &onPass, func(step string, s *runState) {
		if step == stepResolve {
			st = s
		}
	})
	done = make(chan error, 1)
	go func() {
		_, err := run(ctx, root, storeDir, Options{Workers: workers})
		done <- err
	}()
	return done, func() *runState { return st }, func() { once.Do(func() { close(gate) }) }
}

// TestSinkBackpressure stops the writer and checks that every worker then
// waits for a token, not on its send. With W tokens out, W units left
// their bodies: one is in the writer, and the channel holds the others. A
// worker that blocks on its send has not left its body, and the channel
// then holds fewer.
func TestSinkBackpressure(t *testing.T) {
	const workers = 4
	synctest.Test(t, func(t *testing.T) {
		counts := countSteps(t)
		done, state, letGo := stalledRun(t, context.Background(), workers)

		synctest.Wait()
		claims, releases := counts.of(stepResolve)
		if claims != workers || releases != 0 {
			t.Errorf("with the writer stopped, pass 3 has %d claims and %d releases; want exactly %d claims and no release", claims, releases, workers)
		}
		if st := state(); st == nil {
			t.Error("onPass did not run for pass 3")
		} else if got := len(st.units); got != workers-1 {
			t.Errorf("the channel to the writer holds %d units, want %d: with the unit in the writer, every worker left its body and waits for a token", got, workers-1)
		}

		// Positive assertion: the run ends well once the writer goes on.
		letGo()
		if err := <-done; err != nil {
			t.Errorf("run after the writer went on: %v", err)
		}
		synctest.Wait()
	})
}

// TestCancelStopsBlockedWorkers cancels a run while every worker waits for
// a token, and checks that the run returns the cancel error.
func TestCancelStopsBlockedWorkers(t *testing.T) {
	const workers = 4
	synctest.Test(t, func(t *testing.T) {
		counts := countSteps(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done, _, letGo := stalledRun(t, ctx, workers)
		defer letGo()

		synctest.Wait()
		// Positive assertion: the workers were blocked when the cancel came.
		if claims, _ := counts.of(stepResolve); claims != workers {
			t.Fatalf("pass 3 has %d claims before the cancel, want %d", claims, workers)
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("run after the cancel: %v, want context.Canceled", err)
		}
		synctest.Wait()
	})
}

// TestWriterErrorStopsRun checks that the first error of the writer ends
// the run with that error, and that no goroutine of the run outlives it.
func TestWriterErrorStopsRun(t *testing.T) {
	root := goTree(t, 500, false)
	storeDir := filepath.Join(t.TempDir(), "store")
	setHook(t, &batchCap, lowCap)
	commits := 0
	setHook(t, &onCommit, func(graphstore.Commit) error {
		if commits++; commits == 2 {
			return errInjectedCommitFailure
		}
		return nil
	})
	before := goleak.IgnoreCurrent()

	_, err := run(context.Background(), root, storeDir, Options{})
	goleak.VerifyNone(t, before)
	if !errors.Is(err, errInjectedCommitFailure) {
		t.Fatalf("run error = %v, want the error of the second commit", err)
	}
	if commits != 2 {
		t.Errorf("%d commits ran, want the run to stop at the second", commits)
	}
	if left := scratchDirs(t, storeDir); len(left) != 0 {
		t.Errorf("the run left the scratch directories %v", left)
	}
}

// TestWorkerErrorStopsRun injects a failure into one worker of each step
// and checks that the run returns it and leaves nothing behind.
func TestWorkerErrorStopsRun(t *testing.T) {
	for _, step := range []string{stepExtract, stepRoutes, stepStructure, stepResolve} {
		t.Run(step, func(t *testing.T) {
			root := goTree(t, 50, true)
			storeDir := filepath.Join(t.TempDir(), "store")
			var mu sync.Mutex
			claims := 0
			setHook(t, &onClaim, func(s string, file int) error {
				if s != step {
					return nil
				}
				mu.Lock()
				defer mu.Unlock()
				// The second claim of the step fails, so the step had a
				// good claim before the failure.
				if claims++; claims == 2 {
					return fmt.Errorf("%w: injected into %s, file %d", goextract.ErrInternal, s, file)
				}
				return nil
			})
			before := goleak.IgnoreCurrent()

			_, err := run(context.Background(), root, storeDir, Options{Workers: 2})
			goleak.VerifyNone(t, before)
			if !errors.Is(err, goextract.ErrInternal) {
				t.Fatalf("run error = %v, want one that wraps goextract.ErrInternal", err)
			}
			// Positive assertion: the step ran, and the hook failed it.
			if claims < 2 {
				t.Fatalf("the step had %d claims, so the hook did not cause the failure", claims)
			}
			if left := scratchDirs(t, storeDir); len(left) != 0 {
				t.Errorf("the run left the scratch directories %v", left)
			}
			if _, err := os.Lstat(storeDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Lstat(%s) = %v, want no store after a first index that failed", storeDir, err)
			}
		})
	}
}
