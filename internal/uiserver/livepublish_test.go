package uiserver

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// --- Task 1 fixtures ---------------------------------------------------

// writeLiveMeta git-independently writes a real Meta record directly
// through graphstore (never through the full indexer pipeline — this is
// deliberate: these tests need PRECISE control over last_sync_unix_ms and
// commit_sha, which a full indexer.Run call cannot give without racing
// wall-clock timing). It creates repoDir/.codegraph/store if needed,
// opens it as a real Pebble store, writes meta, commits and closes —
// exercising the SAME on-disk shape query.OpenAt/graphstore.Open read
// back, never a stub.
func writeLiveMeta(t *testing.T, repoDir string, meta *schema.Meta) {
	t.Helper()
	storeDir := filepath.Join(repoDir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if err := w.PutMeta(meta); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close: %v", err)
	}
}

// countingCloser wraps an io.Closer and increments *closes exactly once
// per Close call — used to prove the open/close balance test's
// "opens == closes" claim independently of whatever the underlying
// engineCloser itself does.
type countingCloser struct {
	io.Closer
	closes *int64
}

func (c countingCloser) Close() error {
	atomic.AddInt64(c.closes, 1)
	return c.Closer.Close()
}

// withCountingOpenEngine overrides the package's openEngine seam so a
// test can independently count how many times it was called and how many
// times the returned closer's Close was actually invoked. Restores the
// original on t.Cleanup.
func withCountingOpenEngine(t *testing.T) (opens, closes *int64) {
	t.Helper()
	opens = new(int64)
	closes = new(int64)
	orig := openEngine
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		eng, closer, err := orig(start)
		if err != nil {
			return eng, closer, err
		}
		atomic.AddInt64(opens, 1)
		return eng, countingCloser{Closer: closer, closes: closes}, nil
	}
	t.Cleanup(func() { openEngine = orig })
	return opens, closes
}

// withCountingEngineStatus overrides the package's engineStatus seam so a
// test can count how many times (*query.Engine).Status was actually
// invoked, without needing to stub the Engine itself (T-06-40's scoping
// test: Status must be provably NOT called across unchanged wakes, and
// called exactly once across a changed wake, in the same test).
func withCountingEngineStatus(t *testing.T) *int64 {
	t.Helper()
	calls := new(int64)
	orig := engineStatus
	engineStatus = func(e *query.Engine, ctx context.Context) (query.StatusResult, error) {
		atomic.AddInt64(calls, 1)
		return orig(e, ctx)
	}
	t.Cleanup(func() { engineStatus = orig })
	return calls
}

// --- Task 1: the change detector ----------------------------------------

// TestLiveChangeFirstCheckPublishes proves the bootstrap behavior: a
// repository with a real indexed store, checked for the very first time,
// always produces exactly one event — regardless of what liveSignature's
// zero value happens to look like.
func TestLiveChangeFirstCheckPublishes(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)
	ev, changed := d.check(context.Background())
	if !changed {
		t.Fatal("first check: changed = false, want true")
	}
	if ev == nil {
		t.Fatal("first check: event = nil, want a real event")
	}
	if ev.GetGeneration() != 1 {
		t.Fatalf("first check: generation = %d, want 1", ev.GetGeneration())
	}
	if !ev.GetInitialized() {
		t.Fatal("first check against an indexed store: initialized = false, want true")
	}
}

// TestLiveChangeNoEventWhenUnchanged proves D-01's correctness filter:
// a second check against an UNCHANGED last_sync_unix_ms produces no
// event at all.
func TestLiveChangeNoEventWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)
	if _, changed := d.check(context.Background()); !changed {
		t.Fatal("bootstrap check: changed = false, want true (test assumption broken)")
	}

	if ev, changed := d.check(context.Background()); changed {
		t.Fatalf("second check with unchanged last_sync_unix_ms: changed = true (event %+v), want false", ev)
	}
}

// TestLiveChangeEventOnChange proves the positive half of D-01: when
// last_sync_unix_ms genuinely moves, exactly one event is produced,
// carrying a generation strictly greater than the previous one.
func TestLiveChangeEventOnChange(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)
	firstEv, changed := d.check(context.Background())
	if !changed {
		t.Fatal("bootstrap check: changed = false, want true (test assumption broken)")
	}

	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})

	secondEv, changed := d.check(context.Background())
	if !changed {
		t.Fatal("check after last_sync_unix_ms moved: changed = false, want true")
	}
	if secondEv.GetGeneration() <= firstEv.GetGeneration() {
		t.Fatalf("second event generation = %d, want strictly greater than first event generation %d",
			secondEv.GetGeneration(), firstEv.GetGeneration())
	}
}

// TestLiveChangeLockHeldSoftSkip proves T-06-12: a store whose exclusive
// lock is held produces no event, no error escalation, and leaves the
// detector's state untouched — paired with a positive assertion that a
// SUBSEQUENT unlocked check does publish (so a permanently-silent
// detector cannot pass this test).
func TestLiveChangeLockHeldSoftSkip(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)

	orig := openEngine
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		return nil, nil, graphstore.ErrStoreLocked
	}
	if ev, changed := d.check(context.Background()); changed {
		openEngine = orig
		t.Fatalf("check against a locked store: changed = true (event %+v), want false", ev)
	}
	openEngine = orig

	// A subsequent, genuinely unlocked check must still publish — proving
	// the lock-held check did not corrupt or permanently freeze state.
	if _, changed := d.check(context.Background()); !changed {
		t.Fatal("check after the lock was released: changed = false, want true (detector went permanently silent)")
	}
}

// TestLiveChangeNoIndexAtAll proves the "no index at all" behavior: a
// repository with NO .codegraph directory produces exactly one event
// (initialized=false, store_exists=false), and does not thereafter
// republish it every wake.
func TestLiveChangeNoIndexAtAll(t *testing.T) {
	dir := t.TempDir()

	d := newChangeDetector(dir)
	ev, changed := d.check(context.Background())
	if !changed {
		t.Fatal("first check against an un-indexed repo: changed = false, want true")
	}
	if ev.GetInitialized() {
		t.Fatal("un-indexed repo: initialized = true, want false")
	}
	if ev.GetStoreExists() {
		t.Fatal("repo with no .codegraph directory at all: store_exists = true, want false")
	}

	if ev2, changed := d.check(context.Background()); changed {
		t.Fatalf("second check against the SAME un-indexed repo: changed = true (event %+v), want false (must not republish every wake)", ev2)
	}
}

// TestLiveChangeOpenCloseBalance proves SRV-04/T-06-05: every check that
// opens the store also closes it. Asserts BOTH opens == closes AND
// opens >= 2 — an equality alone is satisfied by zero opens, which would
// prove nothing about the store ever actually being touched.
func TestLiveChangeOpenCloseBalance(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	opens, closes := withCountingOpenEngine(t)

	d := newChangeDetector(dir)
	d.check(context.Background())
	d.check(context.Background())
	d.check(context.Background())

	gotOpens := atomic.LoadInt64(opens)
	gotCloses := atomic.LoadInt64(closes)
	if gotOpens < 2 {
		t.Fatalf("opens = %d, want >= 2 (three checks against a real store must open at least twice)", gotOpens)
	}
	if gotOpens != gotCloses {
		t.Fatalf("opens = %d, closes = %d, want equal — a store handle outlived its check", gotOpens, gotCloses)
	}
}

// TestLiveChangeStatusScoped proves T-06-40: the expensive status
// derivation runs ONLY when the signature actually changed. Asserts ZERO
// derivations across 3 unchanged wakes AND exactly ONE across a changed
// wake, in the same test — the zero alone would be satisfied by a
// detector that never ran at all.
func TestLiveChangeStatusScoped(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)
	// Bootstrap check consumes one status derivation of its own — do this
	// BEFORE installing the counter so the counter only measures the
	// window under test.
	if _, changed := d.check(context.Background()); !changed {
		t.Fatal("bootstrap check: changed = false, want true (test assumption broken)")
	}

	calls := withCountingEngineStatus(t)

	for i := 0; i < 3; i++ {
		if _, changed := d.check(context.Background()); changed {
			t.Fatalf("unchanged wake %d: changed = true, want false (test assumption broken)", i)
		}
	}
	if got := atomic.LoadInt64(calls); got != 0 {
		t.Fatalf("status derivations across 3 unchanged wakes = %d, want 0", got)
	}

	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})
	if _, changed := d.check(context.Background()); !changed {
		t.Fatal("check after last_sync_unix_ms moved: changed = false, want true")
	}
	if got := atomic.LoadInt64(calls); got != 1 {
		t.Fatalf("status derivations across 1 changed wake = %d, want exactly 1", got)
	}
}

// TestLiveChangeMalformedCommitSHADegradesToEmpty proves IN-06's
// invariant holds at this exit too (mirroring GetStatus's own commit_sha
// validation): a Meta record carrying a commit SHA that fails
// schema.IsCommitSHA degrades to the empty string on the wire, never an
// unvalidated pass-through.
func TestLiveChangeMalformedCommitSHADegradesToEmpty(t *testing.T) {
	dir := t.TempDir()
	// PutMeta's own write-time validation (internal/indexer/commit.go)
	// never persists a malformed SHA — so to exercise this READ-time
	// degrade path, bypass that discipline by writing directly into the
	// store the same way writeLiveMeta does, but with a hand-crafted
	// malformed value. Meta itself carries no validation at the proto
	// level (only IsCommitSHA at read time enforces the shape).
	writeLiveMeta(t, dir, &schema.Meta{
		SchemaVersion:  schema.SchemaVersion,
		LastSyncUnixMs: 1000,
		CommitSha:      "not-a-real-sha",
	})

	d := newChangeDetector(dir)
	ev, changed := d.check(context.Background())
	if !changed {
		t.Fatal("first check: changed = false, want true")
	}
	if got := ev.GetCommitSha(); got != "" {
		t.Fatalf("commit_sha for a malformed stored SHA = %q, want empty string (must degrade, never pass through unvalidated)", got)
	}
}

// TestLiveChangeUnclassifiedOpenErrorNeverPublishesForever documents the
// defensive fallback for an open failure that is neither
// graphstore.ErrStoreLocked nor query.ErrNotInitialized: the detector
// must still never crash, and must still recover once the underlying
// condition clears — proven the same way the lock-held test proves it.
func TestLiveChangeUnclassifiedOpenErrorNeverPublishesForever(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	d := newChangeDetector(dir)

	orig := openEngine
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		return nil, nil, errors.New("uiserver test: simulated unclassified open failure")
	}
	if ev, changed := d.check(context.Background()); !changed {
		t.Fatal("first check against a simulated open failure: changed = false, want true (an unclassified failure still bootstraps a not-initialized-shaped event)")
	} else if ev.GetInitialized() {
		t.Fatal("simulated open failure: initialized = true, want false")
	}
	openEngine = orig

	if _, changed := d.check(context.Background()); !changed {
		t.Fatal("check after the simulated failure cleared: changed = false, want true (detector went permanently silent)")
	}
}
