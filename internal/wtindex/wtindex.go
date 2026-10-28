// Package wtindex gives a linked git worktree an index of its own, made
// from the index of another working tree of the same repository.
//
// Prepare copies the other index as a pebble checkpoint into a temporary
// directory under the worktree root and syncs the copy with the worktree's
// files, when few enough files differ. Otherwise it returns a pending Build,
// whose Finish builds the index later. Either way the finished index is
// renamed to <worktree>/.codegraph, so a reader never opens one that is not
// complete.
package wtindex

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Limits bound the work Prepare does before it returns, and choose how
// Finish builds the index.
type Limits struct {
	// Foreground is the most changed files Prepare syncs before it
	// returns. It must be at least 1. More changed files leave the build
	// to Finish.
	Foreground int
	// SeededFraction is the largest share of changed files, out of the
	// worktree's files, for which Finish syncs the copied index. Above it,
	// Finish indexes the worktree from scratch.
	SeededFraction float64
}

// DefaultLimits are the limits codegraph serve uses.
var DefaultLimits = Limits{Foreground: 50, SeededFraction: 0.10}

// The reasons a Build is pending.
const (
	ReasonTooManyChanges = "too many files differ from the main checkout's index"
	ReasonMainLocked     = "another codegraph process holds the main checkout's index"
	ReasonMainOutdated   = "the main checkout's index needs a full rebuild"
	ReasonOtherDevice    = "the main checkout's index is on another filesystem"
)

// Build is the outcome of Prepare.
type Build struct {
	// Ready reports that <worktree>/.codegraph exists.
	Ready bool
	// Reason says why the build is pending. It is empty when Ready.
	Reason string
	// Changed and Total count the files that differ from the copied index
	// and the worktree's files. Prepare sets them only when it stopped its
	// sync because more than Limits.Foreground files changed.
	Changed, Total int

	worktreeRoot string
	lim          Limits
	dir          *seedDir
	seeded       bool
	// rebuilt reports that Finish indexed the worktree from scratch.
	rebuilt bool
	// stopped is closed when Finish's build has stopped and its
	// temporary directory is placed or gone.
	stopped chan struct{}
}

// Prepare gives the worktree at worktreeRoot an index made from the index
// of the working tree at mainRoot, when that takes at most lim.Foreground
// changed files. It returns a ready Build when <worktree>/.codegraph
// exists, and a pending Build when the index is left to Finish. It never
// changes mainRoot's index. An error means no index will be made, and the
// temporary directory is gone.
func Prepare(ctx context.Context, worktreeRoot, mainRoot string, lim Limits) (*Build, error) {
	if _, err := os.Lstat(filepath.Join(worktreeRoot, indexDirName)); err == nil {
		return &Build{Ready: true}, nil
	}
	sweep(worktreeRoot)
	dir, err := newSeedDir(worktreeRoot)
	if err != nil {
		return nil, err
	}
	b := &Build{worktreeRoot: worktreeRoot, lim: lim, dir: dir}

	reason, err := b.seed(filepath.Join(mainRoot, indexDirName, storeDirName))
	if err != nil {
		dir.remove()
		return nil, err
	}
	if reason != "" {
		b.Reason = reason
		return b, nil
	}

	_, err = indexer.Sync(worktreeRoot, dir.storePath(), indexer.Options{Quiet: true, MaxChanged: lim.Foreground})
	var tooMany *indexer.ErrTooManyChanges
	switch {
	case errors.As(err, &tooMany):
		b.Reason = ReasonTooManyChanges
		b.Changed, b.Total = tooMany.Changed, tooMany.Total
		return b, nil
	case err != nil:
		dir.remove()
		return nil, err
	}
	if err := b.place(ctx); err != nil {
		return nil, err
	}
	return b, nil
}

// seed checkpoints mainStore into the build's directory. It returns the
// reason the copy cannot be used, or "" when it can.
func (b *Build) seed(mainStore string) (string, error) {
	same, err := sameDevice(mainStore, b.worktreeRoot)
	if err != nil {
		return "", err
	}
	if !same {
		return ReasonOtherDevice, nil
	}
	store := b.dir.storePath()
	if err := graphstore.Checkpoint(mainStore, store); err != nil {
		if errors.Is(err, graphstore.ErrStoreLocked) {
			return ReasonMainLocked, nil
		}
		return "", err
	}
	usable, err := seedUsable(store)
	if err != nil {
		return "", err
	}
	if !usable {
		return ReasonMainOutdated, os.RemoveAll(store)
	}
	b.seeded = true
	return "", nil
}

// seedUsable reports whether a sync of the store at storeDir stays
// incremental: the store has every record a sync needs and this build's
// schema version.
func seedUsable(storeDir string) (bool, error) {
	full, err := indexer.NeedsFullIndex(storeDir)
	if err != nil || full {
		return false, err
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return false, err
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		return false, err
	}
	defer r.Close()
	meta, err := r.GetMeta()
	if err != nil {
		return false, err
	}
	return schema.IsCurrentSchemaVersion(meta), nil
}

// Finish builds a pending index and places it. It syncs the copied index
// when at most lim.SeededFraction of the worktree's files changed, and
// indexes the worktree from scratch otherwise. It returns ctx's error,
// without placing the index, when ctx ends first. Another index that
// appears at the worktree root first wins, and Finish discards its own.
func (b *Build) Finish(ctx context.Context) error {
	if b.Ready {
		return nil
	}
	b.stopped = make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- b.build() }()
	select {
	case err := <-done:
		defer close(b.stopped)
		if err != nil {
			b.dir.remove()
			return err
		}
		return b.place(ctx)
	case <-ctx.Done():
		go func() {
			defer close(b.stopped)
			<-done
			b.dir.remove()
		}()
		return ctx.Err()
	}
}

func (b *Build) build() error {
	store := b.dir.storePath()
	if b.seeded && float64(b.Changed) <= b.lim.SeededFraction*float64(b.Total) {
		_, err := indexer.Sync(b.worktreeRoot, store, indexer.Options{Quiet: true})
		return err
	}
	b.rebuilt = true
	if err := os.RemoveAll(store); err != nil {
		return err
	}
	_, err := indexer.Run(b.worktreeRoot, store, indexer.Options{Quiet: true})
	return err
}

// place moves the build's directory into place, unless ctx has ended. An
// index already at the worktree root counts as placed.
func (b *Build) place(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		b.dir.remove()
		return err
	}
	if err := place(b.dir, b.worktreeRoot); err != nil && !errors.Is(err, ErrIndexExists) {
		return err
	}
	b.Ready, b.Reason, b.dir = true, "", nil
	return nil
}
