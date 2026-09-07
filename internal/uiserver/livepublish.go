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
	"errors"
	"sync"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
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
// the event when changed is true, and changed itself.
//
// query.ErrNotInitialized and any other unclassified openEngine failure
// are folded into the SAME "not initialized" branch, deliberately: the
// publisher must never crash on a filesystem it cannot fully interpret
// (D-01's "never crash" mandate), and GetStatus's own degradedStatus
// (degrade.go) draws exactly the same "answer from filesystem facts once
// open has already failed" line for classifyDegrade's degradeNone case —
// there is no third answer to give here that GetStatus itself would give
// either. graphstore.ErrStoreLocked is the sole exception, classified by
// its exported sentinel via errors.Is (never message text, matching
// classifyOpenError's own discipline) — a re-index in flight is exactly
// when the store is locked, and escalating that into an event would
// defeat the point of live push at the one moment it matters most.
func computeChange(ctx context.Context, repoPath string, prev liveSignature, prevValid bool) (liveSignature, *uiv1.WatchGraphEvent, bool) {
	eng, closer, err := openEngine(repoPath)
	if err != nil {
		if errors.Is(err, graphstore.ErrStoreLocked) {
			// T-06-12: soft outcome, never a crash, never a publish. The
			// caller (changeDetector.check) treats changed=false as
			// "leave my retained state exactly as it was" — returning
			// prev/prevValid unchanged here is what "untouched" means.
			return prev, nil, false
		}

		// query.ErrNotInitialized (no ancestor .codegraph/ found) or any
		// other unclassified failure: mirror degradedStatus's filesystem
		// read — query.ResolveCodegraphDir succeeds independently of
		// whatever caused openEngine to fail, since it only os.Stats the
		// .codegraph/ directory.
		storeExists := false
		if _, resolveErr := query.ResolveCodegraphDir(repoPath); resolveErr == nil {
			storeExists = true
		}
		sig := liveSignature{openFailed: true, storeExists: storeExists}
		if prevValid && sig == prev {
			return prev, nil, false
		}
		return sig, &uiv1.WatchGraphEvent{
			Initialized: false,
			StoreExists: storeExists,
		}, true
	}
	defer closer.Close()

	// SRV-04/T-06-05: this defer is the ENTIRE store-handle lifetime for
	// this check. Nothing below retains eng, the Reader, or the store
	// past this function's return.

	meta, metaErr := eng.IndexMeta()
	if metaErr != nil {
		// A read failure on an already-opened engine — never escalate;
		// retry next wake with state untouched, same soft-outcome shape
		// as the lock-held branch above.
		return prev, nil, false
	}

	sig := liveSignature{hasMeta: meta != nil}
	if meta != nil {
		sig.lastSync = meta.GetLastSyncUnixMs()
	}
	if prevValid && sig == prev {
		// D-01/T-06-40: the correctness filter. Pebble's own compaction,
		// WAL rotation and MANIFEST churn produce wakes with nothing
		// changed, and the expensive status derivation below NEVER runs
		// for them.
		return prev, nil, false
	}

	// Only reached when the signature actually changed (or this is the
	// bootstrap check) — ORDER MATTERS: IndexMeta (a direct reader
	// lookup) ran first and decided whether to proceed at all;
	// engineStatus's filesystem-and-graph scans are strictly more
	// expensive and run only now.
	result, statusErr := engineStatus(eng, ctx)
	if statusErr != nil {
		return prev, nil, false
	}

	// Reuses the SAME inputs statusToProto (handlers.go) maps onto
	// GetStatusResponse — result plus IndexMeta's commit SHA — so there
	// is one mapping from engine state onto these field names, not two;
	// the destination message type differs (WatchGraphEvent vs
	// GetStatusResponse) but the source computation does not.
	//
	// IN-06: schema.IsCommitSHA is applied here too, exactly as
	// GetStatus applies it — a commit SHA that leaves this server is
	// well-formed at every exit, not only GetStatus's.
	commitSHA, ok := schema.IndexedCommitSHA(meta)
	if ok && !schema.IsCommitSHA(commitSHA) {
		commitSHA = ""
	}

	// indexing_in_progress mirrors GetStatusResponse.indexing_in_progress
	// for classifyStatus's benefit, but — per the plan's own documented
	// limitation — this detector never observes an in-flight index: while
	// a writer holds the store's exclusive lock, computeChange takes the
	// ErrStoreLocked branch above and emits nothing; by the time the
	// store opens again, indexing has already finished. So this field
	// reads false in every live event this detector ever produces. It is
	// carried anyway because the wire shape mirrors GetStatusResponse's
	// field set exactly (D-07) and because Status's own IndexHealth
	// state derivation is the one true source for it if that ever
	// changes — nothing downstream may treat "false here" as "no index
	// is running".
	return sig, &uiv1.WatchGraphEvent{
		Initialized:        result.Initialized,
		Stale:              result.Stale,
		StoreExists:        true,
		IndexingInProgress: false,
		CommitSha:          commitSHA,
	}, true
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
// changed is true.
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

// liveRegistry is the bounded, coalescing fan-out registry (D-04): every
// subscriber holds its own capacity-1 channel, guarded under registry's
// own mu, and a full buffer is drained-then-replaced rather than
// blocking the publisher or dropping the newest event. No message-queue
// dependency — a handful of browser tabs does not justify one, and the
// coalescing semantics are custom regardless of library choice (see
// 06-PATTERNS.md's own "No Analog Found" entry for this type: nothing in
// this codebase holds a live multi-subscriber channel map today).
//
// sendCount and the subscriber map are STRUCTURALLY separate storage
// (T-06-10): this repository has already shipped one bug (internal/mcp's
// pendingWriter, server.go:466) where server-initiated writes corrupted
// a counter meaning client-initiated pending state. The streaming rpc
// has the identical shape — server-initiated event sends alongside
// client-initiated subscriber lifecycle — so Publish never touches subs'
// length and Subscribe/unsubscribe never touch sendCount. A full read of
// every non-test file in this package (at the time this file was
// written) found no pre-existing counter of that kind to inherit; that
// verdict and its evidence are recorded in 06-06-SUMMARY.md per
// 06-CONTEXT.md's "Criterion 3's recorded verdict" requirement.
type liveRegistry struct {
	mu      sync.Mutex
	subs    map[uint64]chan *uiv1.WatchGraphEvent
	nextID  uint64
	current *uiv1.WatchGraphEvent

	sendCount int64 // placeholder — implemented in the GREEN commit of Task 2

	done     chan struct{}
	stopOnce sync.Once
}

// newLiveRegistry returns an empty, unstarted registry. Placeholder —
// implemented in the GREEN commit of Task 2.
func newLiveRegistry() *liveRegistry {
	return &liveRegistry{}
}

// Subscribe registers a new subscriber and returns a receive-only
// channel plus an unsubscribe func. Placeholder — implemented in the
// GREEN commit of Task 2.
func (r *liveRegistry) Subscribe(ctx context.Context) (<-chan *uiv1.WatchGraphEvent, func()) {
	return nil, func() {}
}

// Publish fans ev out to every current subscriber. Placeholder —
// implemented in the GREEN commit of Task 2.
func (r *liveRegistry) Publish(ev *uiv1.WatchGraphEvent) {
}

// Stop closes every subscriber channel and releases the registry.
// Placeholder — implemented in the GREEN commit of Task 2.
func (r *liveRegistry) Stop() {
}

// SendCount reports the number of server-initiated event sends.
// Placeholder — implemented in the GREEN commit of Task 2.
func (r *liveRegistry) SendCount() int64 {
	return 0
}

// SubscriberCount reports the number of currently-registered
// subscribers. Placeholder — implemented in the GREEN commit of Task 2.
func (r *liveRegistry) SubscriberCount() int {
	return 0
}

// Current returns the registry's last-known event, or nil before the
// first Publish. Placeholder — implemented in the GREEN commit of
// Task 2.
func (r *liveRegistry) Current() *uiv1.WatchGraphEvent {
	return nil
}
