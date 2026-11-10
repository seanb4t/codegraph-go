package indexer

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// The names of the steps of a full index that run on forEachFile. The
// hooks below get one of them, so a test can tell the steps apart.
const (
	stepExtract   = "pass 1"
	stepRoutes    = "route detection"
	stepStructure = "pass 2"
	stepResolve   = "pass 3"
)

// These variables give the tests of this package control over a run, in
// the style of gitExecLookPath. Production code never sets them.
var (
	// claimOrder maps the counter value of a claim, from 0 to n-1, to the
	// position that the claim takes. Nil means the counter value itself. A
	// test sets a permutation to change the order in which workers take
	// files.
	claimOrder func(counter, n int) int

	// onClaim runs when a worker took a token and claimed a file, and
	// onRelease when that token goes back. file is the index of the file
	// among the files of the run. If onClaim returns an error, the body of
	// that file does not run, and the loop fails with that error.
	onClaim   func(step string, file int) error
	onRelease func(step string, file int, state *runState)

	// onPass runs at the start of each step, with the state of the run.
	onPass func(step string, state *runState)

	// onExchange runs after the new store took the place of the live
	// store, and before the run removes its scratch directory.
	onExchange func()

	// batchCap is the cap that run passes to the bounded writer, and
	// onCommit the commit callback that it passes.
	batchCap = graphstore.DefaultBatchCap
	onCommit func(graphstore.Commit) error
)

// claimLoop names the step that a forEachFile call runs, for the hooks.
type claimLoop struct {
	step  string
	state *runState
	// files maps a position of the loop to a file index, when the loop
	// runs over a part of the files. Nil means the position itself.
	files []int
}

func (l claimLoop) fileOf(position int) int {
	if l.files != nil {
		return l.files[position]
	}
	return position
}

// forEachFile runs body once for each position from 0 to n-1, on up to
// workers goroutines. It is the one claim loop of a full index: every pass
// runs on it, and so does Extract.
//
// A worker takes a token before it claims a position, and the token goes
// back when release runs. So at most workers positions hold a token at any
// time, whatever the bodies do after they return. body gets the release of
// its position. A body that keeps working state past its return, such as
// one that hands a unit to a writer, passes release on and returns without
// calling it. Every other body calls it when it is done. release acts once,
// and the loop calls it for a body that returns an error, so a failed
// position leaks no token.
//
// workers of 0 or below means DefaultWorkers. The loop starts no more
// goroutines than n, so a large worker count costs nothing for a small
// run. setup runs once on each goroutine and returns what body gets as w,
// with a cleanup for the end of the goroutine: the parser cache of a
// worker, for example.
//
// The first error stops the loop: it cancels the context that the other
// bodies and token waits see, and forEachFile returns that error once
// every goroutine ended.
func forEachFile[W any](
	ctx context.Context, loop claimLoop, n, workers int,
	setup func() (w W, cleanup func(), err error),
	body func(ctx context.Context, w W, position int, release func()) error,
) error {
	if n == 0 {
		return nil
	}
	if workers <= 0 {
		workers = DefaultWorkers()
	}
	tokens := semaphore.NewWeighted(int64(workers))
	var next atomic.Int64

	g, ctx := errgroup.WithContext(ctx)
	for range min(workers, n) {
		g.Go(func() error {
			w, cleanup, err := setup()
			if err != nil {
				return err
			}
			defer cleanup()
			for {
				if err := tokens.Acquire(ctx, 1); err != nil {
					return err
				}
				counter := int(next.Add(1)) - 1
				if counter >= n {
					tokens.Release(1)
					return nil
				}
				position := counter
				if claimOrder != nil {
					position = claimOrder(counter, n)
				}
				file := loop.fileOf(position)
				var once sync.Once
				release := func() {
					once.Do(func() {
						tokens.Release(1)
						if onRelease != nil {
							onRelease(loop.step, file, loop.state)
						}
					})
				}
				if onClaim != nil {
					if err := onClaim(loop.step, file); err != nil {
						release()
						return err
					}
				}
				if err := body(ctx, w, position, release); err != nil {
					release()
					return err
				}
			}
		})
	}
	return g.Wait()
}
