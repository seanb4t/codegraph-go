package indexer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
)

// tokenPeak counts the positions that hold a token, over every step of a
// run or over one forEachFile loop.
type tokenPeak struct {
	mu       sync.Mutex
	current  int
	peak     int
	claims   map[int]int
	releases map[int]int
	// reach is the count that the first claims wait for, or 0 for no
	// wait. With it set, a loop that lets reach positions in at once
	// always gets there, and one that does not stops at its first claims.
	reach   int
	reached chan struct{}
}

func newTokenPeak(reach int) *tokenPeak {
	return &tokenPeak{claims: map[int]int{}, releases: map[int]int{}, reach: reach, reached: make(chan struct{})}
}

func (p *tokenPeak) claim(file int) {
	p.mu.Lock()
	p.current++
	p.peak = max(p.peak, p.current)
	p.claims[file]++
	if p.reach > 0 && p.current == p.reach {
		select {
		case <-p.reached:
		default:
			close(p.reached)
		}
	}
	p.mu.Unlock()
	if p.reach > 0 {
		<-p.reached
	}
}

func (p *tokenPeak) release(file int) {
	p.mu.Lock()
	p.current--
	p.releases[file]++
	p.mu.Unlock()
}

// TestDefaultWorkersFollowsGOMAXPROCS checks the worker count of a full
// index: a count of 0 or below means GOMAXPROCS, and a count far above the
// file count starts no more workers than there are files.
func TestDefaultWorkersFollowsGOMAXPROCS(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(3))
	for _, tc := range []struct {
		workers, files, wantPeak int
	}{
		{-1, 30, 3},
		{0, 30, 3},
		{1, 30, 1},
		{1 << 40, 3, 3},
	} {
		t.Run(fmt.Sprintf("workers %d over %d files", tc.workers, tc.files), func(t *testing.T) {
			root := goTree(t, tc.files, false)
			// One counter for each step: the steps run one after another,
			// and each must reach the peak on its own.
			peaks := map[string]*tokenPeak{}
			var mu sync.Mutex
			peakOf := func(step string) *tokenPeak {
				mu.Lock()
				defer mu.Unlock()
				if peaks[step] == nil {
					peaks[step] = newTokenPeak(tc.wantPeak)
				}
				return peaks[step]
			}
			setHook(t, &onClaim, func(step string, file int) error {
				peakOf(step).claim(file)
				return nil
			})
			setHook(t, &onRelease, func(step string, file int, _ *runState) { peakOf(step).release(file) })

			if _, err := run(context.Background(), root, filepath.Join(t.TempDir(), "store"), Options{Workers: tc.workers}); err != nil {
				t.Fatalf("run: %v", err)
			}
			for _, step := range []string{stepExtract, stepStructure, stepResolve} {
				p := peaks[step]
				if p == nil {
					t.Fatalf("%s claimed no file", step)
				}
				// Positive assertion: the step reached its peak, and went
				// no higher.
				if p.peak != tc.wantPeak {
					t.Errorf("%s: at most %d files held a token at once, want exactly %d", step, p.peak, tc.wantPeak)
				}
			}
		})
	}
}

var errBodyFailed = errors.New("the body of this position fails")

// TestForEachFileProperty checks the claim loop on its own, with no store:
// the token bound, the number of workers, one run for each position, and
// one release for each claim, also when bodies fail.
func TestForEachFileProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		workers := rapid.IntRange(-1, 64).Draw(t, "workers")
		n := rapid.IntRange(0, 200).Draw(t, "n")
		order := rapid.Permutation(positions(n)).Draw(t, "claimOrder")
		// For each failing position, whether its body calls release
		// before it returns the error.
		failing := map[int]bool{}
		if n > 0 {
			for range rapid.IntRange(0, 3).Draw(t, "failures") {
				failing[rapid.IntRange(0, n-1).Draw(t, "failing")] = rapid.Bool().Draw(t, "releaseFirst")
			}
		}
		resolved := workers
		if resolved <= 0 {
			resolved = DefaultWorkers()
		}
		bound := min(n, resolved)

		peak := newTokenPeak(0)
		var mu sync.Mutex
		ran := map[int]int{}
		started := 0
		origOrder, origClaim, origRelease := claimOrder, onClaim, onRelease
		claimOrder = func(counter, _ int) int { return order[counter] }
		onClaim = func(_ string, file int) error { peak.claim(file); return nil }
		onRelease = func(_ string, file int, _ *runState) { peak.release(file) }
		defer func() { claimOrder, onClaim, onRelease = origOrder, origClaim, origRelease }()

		err := forEachFile(context.Background(), claimLoop{step: "property"}, n, workers,
			func() (struct{}, func(), error) {
				mu.Lock()
				started++
				mu.Unlock()
				return struct{}{}, func() {}, nil
			},
			func(_ context.Context, _ struct{}, position int, release func()) error {
				mu.Lock()
				ran[position]++
				mu.Unlock()
				if releaseFirst, fails := failing[position]; fails {
					if releaseFirst {
						release()
					}
					return fmt.Errorf("position %d: %w", position, errBodyFailed)
				}
				release()
				return nil
			})

		if peak.peak > bound {
			t.Fatalf("%d positions held a token at once, above the bound %d", peak.peak, bound)
		}
		if len(failing) == 0 {
			if err != nil {
				t.Fatalf("forEachFile: %v", err)
			}
			if started != bound {
				t.Fatalf("%d workers started, want %d", started, bound)
			}
			for position := range n {
				if ran[position] != 1 {
					t.Fatalf("position %d ran %d times, want once", position, ran[position])
				}
			}
		} else if !errors.Is(err, errBodyFailed) {
			t.Fatalf("forEachFile error = %v, want the error of a failing position", err)
		}
		for position, claims := range peak.claims {
			if claims != 1 || peak.releases[position] != 1 {
				t.Fatalf("position %d: %d claims and %d releases, want one of each", position, claims, peak.releases[position])
			}
		}
		if len(peak.releases) != len(peak.claims) {
			t.Fatalf("%d positions were released and %d claimed", len(peak.releases), len(peak.claims))
		}
	})
}

func positions(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// TestForEachFileReleaseOnce checks that release acts once: a body that
// calls it twice gives one token back, not two.
func TestForEachFileReleaseOnce(t *testing.T) {
	peak := newTokenPeak(0)
	setHook(t, &onClaim, func(_ string, file int) error { peak.claim(file); return nil })
	setHook(t, &onRelease, func(_ string, file int, _ *runState) { peak.release(file) })

	ran := 0
	err := forEachFile(context.Background(), claimLoop{step: "release once"}, 3, 1, noSetup,
		func(_ context.Context, _ struct{}, _ int, release func()) error {
			ran++
			release()
			release()
			return nil
		})
	if err != nil {
		t.Fatalf("forEachFile: %v", err)
	}
	// Positive assertion: every file ran.
	if ran != 3 {
		t.Fatalf("%d bodies ran, want 3", ran)
	}
	if peak.peak != 1 {
		t.Errorf("%d files held a token at once with one worker, want 1: a second release gave a second token back", peak.peak)
	}
	for file := range 3 {
		if peak.releases[file] != 1 {
			t.Errorf("file %d: onRelease ran %d times, want once", file, peak.releases[file])
		}
	}
}

// TestSyncDuringFullIndexFails checks that a full index is the only writer
// of its store: a sync that starts during the run fails, and the store
// then holds the graph of the index.
func TestSyncDuringFullIndexFails(t *testing.T) {
	root, storeDir := indexedTree(t)
	generation := readMeta(t, storeDir).GetCoverageGeneration()
	writeFile(t, filepath.Join(root, "pkg", "added.go"), "package pkg\n\nfunc Added() {}\n")

	var syncErr error
	synced := false
	setHook(t, &onPass, func(step string, _ *runState) {
		if step == stepExtract {
			_, syncErr = Sync(root, storeDir, Options{})
			synced = true
		}
	})
	if _, err := run(context.Background(), root, storeDir, Options{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !synced {
		t.Fatal("the hook did not run a sync")
	}
	if !errors.Is(syncErr, graphstore.ErrStoreLocked) {
		t.Fatalf("Sync during a full index = %v, want ErrStoreLocked", syncErr)
	}
	onPass = nil
	if got := readMeta(t, storeDir).GetCoverageGeneration(); got != generation+1 {
		t.Errorf("coverage generation = %d, want %d: one more than before the index", got, generation+1)
	}
	if !bytes.Equal(exportNormalized(t, storeDir, true), freshExport(t, root)) {
		t.Errorf("the store does not hold the graph of the full index")
	}
}

// TestReadDuringFullIndexSeesOldGraph checks that a query during a full
// index reads the previous graph, whole.
func TestReadDuringFullIndexSeesOldGraph(t *testing.T) {
	root, storeDir := indexedTree(t)
	before := exportNormalized(t, storeDir, false)
	writeFile(t, filepath.Join(root, "pkg", "added.go"), "package pkg\n\nfunc Added() {}\n")

	var during []byte
	setHook(t, &onPass, func(step string, _ *runState) {
		if step == stepResolve {
			during = exportNormalized(t, storeDir, false)
		}
	})
	if _, err := run(context.Background(), root, storeDir, Options{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if during == nil {
		t.Fatal("the hook did not read the store")
	}
	if !bytes.Equal(during, before) {
		t.Errorf("a read at the start of pass 3 did not see the graph from before the index")
	}
	onPass = nil
	// Positive assertion: the index did change the graph.
	if bytes.Equal(exportNormalized(t, storeDir, false), before) {
		t.Errorf("the store is unchanged after the index, so the read above proves nothing")
	}
}
