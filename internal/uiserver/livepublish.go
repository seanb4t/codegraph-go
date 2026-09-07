// Package uiserver: livepublish.go is the server-side engine of live push
// (Phase 6, LIV-01/LIV-03): a store watcher that learns a re-index
// happened, a change detector that decides whether that is worth telling
// anyone, and a bounded, coalescing fan-out registry that hands the news
// to every open stream without letting a slow reader hurt a fast one.
//
// This file has NO HTTP or Connect dependency of any kind — 06-04's
// handler consumes livePublisher, not the other way round. Every type and
// function here is unexported: the handler that wires this into the wire
// layer is 06-04's job, not this file's.
package uiserver

import (
	"context"
	"sync"

	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// engineStatus is (*query.Engine).Status indirected behind a
// package-level func var — the same test-only control-seam shape
// openEngine (handlers.go) already established in this package — so
// tests can count invocations without a live store. T-06-40's scoping
// test requires proving Status is NEVER derived across N unchanged
// wakes and IS derived exactly once across a changed wake, in the same
// test; that is only observable by intercepting this call, since
// (*query.Engine).Status is a concrete method with no other seam.
// Production behavior is unchanged: the var defaults to
// (*query.Engine).Status and has no exported setter.
var engineStatus = (*query.Engine).Status

// liveSignature is the change detector's own comparable state (D-01):
// what the detector observed on its LAST check, deliberately never
// carried on the wire (D-07 keeps Meta.last_sync_unix_ms server-side
// only). Two liveSignature values compare equal with plain `==` — every
// field is a plain comparable scalar on purpose, so "did anything change"
// is a single struct comparison.
//
//   - openFailed true means openEngine itself failed for a reason other
//     than a held lock (query.ErrNotInitialized, or any other
//     unclassified error — see computeChange's doc comment for why the
//     two are folded together). storeExists is only meaningful in this
//     case.
//   - openFailed false means the engine opened successfully; hasMeta
//     records whether Engine.IndexMeta() returned a record at all, and
//     lastSync is only meaningful when hasMeta is true.
//
// A store whose lock is held (graphstore.ErrStoreLocked) produces NO
// liveSignature at all — computeChange returns the caller's own previous
// signature unchanged for that case, per D-01's "never crash, retry next
// wake" contract.
type liveSignature struct {
	openFailed  bool
	storeExists bool
	hasMeta     bool
	lastSync    int64
}

// computeChange performs ONE open/read/close cycle (SRV-04, T-06-05):
// open via the package's openEngine seam, defer the returned closer's
// Close, read IndexMeta, and decide — never a second query.OpenAt call
// site, and never a store handle retained past this single call.
//
// prevValid distinguishes "never checked before" (always changed, the
// bootstrap case) from a real prior signature that happens to equal the
// liveSignature zero value (an opened-but-never-indexed store, hasMeta
// false, lastSync 0) — collapsing these into one "prevValid" bit is
// necessary because the zero value IS a legitimate signature, not merely
// a sentinel.
//
// Returns the new signature (retained by the caller — changeDetector —
// as the next call's prev, regardless of whether an event was produced),
// the event when changed is true, and changed itself. This is a
// placeholder — implemented in the GREEN commit of Task 1.
func computeChange(ctx context.Context, repoPath string, prev liveSignature, prevValid bool) (liveSignature, *uiv1.WatchGraphEvent, bool) {
	return prev, nil, false
}

// changeDetector wraps computeChange with its own retained state (the
// last observed liveSignature) and the monotonic generation counter
// (incremented once per emitted event, never on a no-op check). mu
// serializes check end-to-end — including the open/read/close cycle
// inside computeChange — so two debounced flushes racing each other (the
// Debouncer's own doc comment documents this as a rare but real
// possibility: Add() can arm a new timer while a fire() already in
// flight has not yet returned) can never interleave two publishes out of
// generation order.
type changeDetector struct {
	repoPath string

	mu    sync.Mutex
	sig   liveSignature
	valid bool
	gen   int64
}

// newChangeDetector returns a changeDetector with no prior state — its
// first check is always "changed" (the bootstrap case computeChange's
// prevValid parameter exists for).
func newChangeDetector(repoPath string) *changeDetector {
	return &changeDetector{repoPath: repoPath}
}

// check runs one computeChange cycle against this detector's retained
// state, updating that state and the generation counter only when
// changed is true. Placeholder — implemented in the GREEN commit of
// Task 1.
func (d *changeDetector) check(ctx context.Context) (*uiv1.WatchGraphEvent, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	next, ev, changed := computeChange(ctx, d.repoPath, d.sig, d.valid)
	if !changed {
		return nil, false
	}
	d.sig = next
	d.valid = true
	d.gen++
	ev.Generation = d.gen
	return ev, true
}
