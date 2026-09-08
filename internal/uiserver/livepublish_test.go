package uiserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
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

	// Task 3's tests run a REAL live publisher concurrently checking this
	// same store in the background — a transient ErrStoreLocked collision
	// between the test's own write and the publisher's own debounced
	// check is exactly the condition graphstore.Open's own bounded retry
	// exists for (openLockRetryAttempts/openLockRetryBackoff), and under
	// -race's instrumentation overhead that budget alone is sometimes not
	// enough. Retry here too, at the fixture level, rather than treating
	// an expected transient collision as a fixture failure.
	var store graphstore.GraphStore
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for {
		store, err = graphstore.Open(storeDir)
		if err == nil || !errors.Is(err, graphstore.ErrStoreLocked) || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
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

// --- Task 2: the fan-out registry ---------------------------------------

// recvWithTimeout receives one value from ch, failing the test if nothing
// arrives within timeout. Returns the received value and whether the
// channel was open (ok).
func recvWithTimeout(t *testing.T, ch <-chan *uiv1.WatchGraphEvent, timeout time.Duration) (*uiv1.WatchGraphEvent, bool) {
	t.Helper()
	select {
	case ev, ok := <-ch:
		return ev, ok
	case <-time.After(timeout):
		t.Fatal("recvWithTimeout: nothing received within timeout")
		return nil, false
	}
}

// assertNothingReceived asserts ch delivers nothing within a short
// window — used to prove a channel was NOT touched (e.g. after
// unsubscribe).
func assertNothingReceived(t *testing.T, ch <-chan *uiv1.WatchGraphEvent, window time.Duration) {
	t.Helper()
	select {
	case ev, ok := <-ch:
		t.Fatalf("assertNothingReceived: received (event=%+v, ok=%v), want nothing", ev, ok)
	case <-time.After(window):
	}
}

// TestLiveRegistrySubscribeReturnsIndependentChannels proves subscribing
// twice yields two independent channels: a publish reaches both, and
// each is drained independently.
func TestLiveRegistrySubscribeReturnsIndependentChannels(t *testing.T) {
	reg := newLiveRegistry()
	ch1, unsub1 := reg.Subscribe(context.Background())
	defer unsub1()
	ch2, unsub2 := reg.Subscribe(context.Background())
	defer unsub2()

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 1})

	ev1, ok1 := recvWithTimeout(t, ch1, time.Second)
	if !ok1 || ev1.GetGeneration() != 1 {
		t.Fatalf("ch1: got (%+v, %v), want generation 1, ok true", ev1, ok1)
	}
	ev2, ok2 := recvWithTimeout(t, ch2, time.Second)
	if !ok2 || ev2.GetGeneration() != 1 {
		t.Fatalf("ch2: got (%+v, %v), want generation 1, ok true", ev2, ok2)
	}
}

// TestLiveRegistrySubscribeSeedsCurrentState proves the seeding contract:
// a brand-new subscriber observes exactly the publisher's CURRENT state
// with ZERO publishes having occurred since Subscribe returned — the
// positive assertion (the observed generation equals the registry's own
// reported current generation) distinguishes "received the real current
// state" from "received a zero value".
func TestLiveRegistrySubscribeSeedsCurrentState(t *testing.T) {
	reg := newLiveRegistry()
	reg.Publish(&uiv1.WatchGraphEvent{Generation: 7})

	ch, unsub := reg.Subscribe(context.Background())
	defer unsub()

	ev, ok := recvWithTimeout(t, ch, time.Second)
	if !ok {
		t.Fatal("seeded channel: ok = false, want true")
	}
	want := reg.Current().GetGeneration()
	if want != 7 {
		t.Fatalf("test assumption broken: reg.Current().GetGeneration() = %d, want 7", want)
	}
	if ev.GetGeneration() != want {
		t.Fatalf("seeded event generation = %d, want %d (the registry's own current generation)", ev.GetGeneration(), want)
	}

	// No publish happened between Subscribe and the receive above — the
	// channel had exactly one thing in it (the seed), so a second receive
	// with no intervening Publish must find nothing.
	assertNothingReceived(t, ch, 50*time.Millisecond)
}

// TestLiveRegistryTwoSubscribersSeeSameCurrentGeneration proves two
// subscribers created at different times both observe the same current
// generation.
func TestLiveRegistryTwoSubscribersSeeSameCurrentGeneration(t *testing.T) {
	reg := newLiveRegistry()
	reg.Publish(&uiv1.WatchGraphEvent{Generation: 3})

	ch1, unsub1 := reg.Subscribe(context.Background())
	defer unsub1()
	ev1, _ := recvWithTimeout(t, ch1, time.Second)

	ch2, unsub2 := reg.Subscribe(context.Background())
	defer unsub2()
	ev2, _ := recvWithTimeout(t, ch2, time.Second)

	if ev1.GetGeneration() != 3 || ev2.GetGeneration() != 3 {
		t.Fatalf("ev1.Generation = %d, ev2.Generation = %d, want both 3", ev1.GetGeneration(), ev2.GetGeneration())
	}
}

// TestLiveRegistryStopClosesSubscriberChannels proves Stop closes every
// subscriber channel: a consumer blocked on receive observes a clean
// close (ok=false) rather than hanging, paired with a positive assertion
// that the same channel delivered at least one real event before Stop.
func TestLiveRegistryStopClosesSubscriberChannels(t *testing.T) {
	reg := newLiveRegistry()
	ch, _ := reg.Subscribe(context.Background())

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 1})
	if _, ok := recvWithTimeout(t, ch, time.Second); !ok {
		t.Fatal("receive before Stop: ok = false, want true (a real event must have been delivered first)")
	}

	reg.Stop()

	if ev, ok := recvWithTimeout(t, ch, time.Second); ok {
		t.Fatalf("receive after Stop: ok = true (event %+v), want false (clean close)", ev)
	}
}

// TestLiveRegistryPublishDeliversToEmptyBuffer proves the base case:
// publishing to a subscriber whose buffer is empty delivers that event.
func TestLiveRegistryPublishDeliversToEmptyBuffer(t *testing.T) {
	reg := newLiveRegistry()
	ch, unsub := reg.Subscribe(context.Background())
	defer unsub()

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 5})

	ev, ok := recvWithTimeout(t, ch, time.Second)
	if !ok || ev.GetGeneration() != 5 {
		t.Fatalf("got (%+v, %v), want generation 5, ok true", ev, ok)
	}
}

// TestLiveRegistryPublishCoalescesFullBuffer proves D-04: publishing to a
// subscriber whose buffer is FULL replaces the unsent event with the
// newer one. The subscriber then reads the NEWER generation, never the
// older — asserted by generation value, not by count.
func TestLiveRegistryPublishCoalescesFullBuffer(t *testing.T) {
	reg := newLiveRegistry()
	ch, unsub := reg.Subscribe(context.Background())
	defer unsub()

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 1}) // fills the empty buffer
	reg.Publish(&uiv1.WatchGraphEvent{Generation: 2}) // buffer full: must replace, not block or drop-newest

	ev, ok := recvWithTimeout(t, ch, time.Second)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if ev.GetGeneration() != 2 {
		t.Fatalf("received generation = %d, want 2 (the newer event, not the coalesced-away 1)", ev.GetGeneration())
	}
}

// TestLiveRegistryPublishNeverBlocksNonReadingSubscriber proves D-04's
// non-blocking guarantee under a realistic mixed-subscriber scenario:
// one subscriber never reads, one reads continuously. All N publishes
// complete (Publish itself never blocks), and the reading subscriber
// observes at least 2 events with its final observed generation equal to
// the last published one — the floor rules out "nothing happened", and
// the equality rules out a stalled/incorrect coalesce.
func TestLiveRegistryPublishNeverBlocksNonReadingSubscriber(t *testing.T) {
	reg := newLiveRegistry()

	nonReadingCh, nonReadingUnsub := reg.Subscribe(context.Background())
	defer nonReadingUnsub()
	_ = nonReadingCh // deliberately never read

	readingCh, readingUnsub := reg.Subscribe(context.Background())
	defer readingUnsub()

	var mu sync.Mutex
	var received []int64
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for ev := range readingCh {
			mu.Lock()
			received = append(received, ev.GetGeneration())
			mu.Unlock()
		}
	}()

	const n = int64(500)
	publishDone := make(chan struct{})
	go func() {
		defer close(publishDone)
		for i := int64(1); i <= n; i++ {
			reg.Publish(&uiv1.WatchGraphEvent{Generation: i})
			runtime.Gosched()
		}
	}()

	select {
	case <-publishDone:
	case <-time.After(5 * time.Second):
		t.Fatal("publishing did not complete within 5s — Publish must never block")
	}

	reg.Stop()
	select {
	case <-readerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("reader goroutine did not observe the channel close within 5s")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(received) < 2 {
		t.Fatalf("reading subscriber observed %d events, want >= 2", len(received))
	}
	if got := received[len(received)-1]; got != n {
		t.Fatalf("final observed generation = %d, want %d (the last published generation)", got, n)
	}
}

// TestLiveRegistryUnsubscribeRemovesSubscriber proves unsubscribing
// removes the subscriber from the registry: the registry's subscriber
// count returns to its prior value, and a subsequent publish does not
// touch the removed channel.
func TestLiveRegistryUnsubscribeRemovesSubscriber(t *testing.T) {
	reg := newLiveRegistry()
	before := reg.SubscriberCount()

	ch, unsub := reg.Subscribe(context.Background())
	if got := reg.SubscriberCount(); got != before+1 {
		t.Fatalf("SubscriberCount after Subscribe = %d, want %d", got, before+1)
	}

	unsub()
	if got := reg.SubscriberCount(); got != before {
		t.Fatalf("SubscriberCount after unsubscribe = %d, want %d (back to prior value)", got, before)
	}

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 99})
	if ev, ok := recvWithTimeout(t, ch, 100*time.Millisecond); ok {
		t.Fatalf("removed channel received (event=%+v, ok=%v) after a subsequent publish, want closed/nothing", ev, ok)
	}
}

// TestLiveRegistryContextCancelRemovesSubscriber proves cancelling a
// subscriber's context removes it exactly as an explicit unsubscribe
// would.
func TestLiveRegistryContextCancelRemovesSubscriber(t *testing.T) {
	reg := newLiveRegistry()
	before := reg.SubscriberCount()

	ctx, cancel := context.WithCancel(context.Background())
	_, unsub := reg.Subscribe(ctx)
	defer unsub()

	if got := reg.SubscriberCount(); got != before+1 {
		t.Fatalf("SubscriberCount after Subscribe = %d, want %d", got, before+1)
	}

	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if reg.SubscriberCount() == before {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("SubscriberCount after context cancellation = %d, want %d within 2s", reg.SubscriberCount(), before)
}

// TestLiveRegistryCounterSeparation proves T-06-10's recorded verdict:
// the count of server-initiated event sends and the count of live
// client-initiated subscribers are two distinct values that never share
// storage. Both directions are asserted, and each counter is non-zero at
// the point it is compared, so neither direction passes vacuously.
func TestLiveRegistryCounterSeparation(t *testing.T) {
	reg := newLiveRegistry()

	ch, unsub := reg.Subscribe(context.Background())
	defer unsub()
	if got := reg.SendCount(); got != 0 {
		t.Fatalf("SendCount after Subscribe = %d, want 0 (subscribing must never touch the send counter)", got)
	}

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 1})
	recvWithTimeout(t, ch, time.Second)
	sendsAfterOnePublish := reg.SendCount()
	if sendsAfterOnePublish == 0 {
		t.Fatal("SendCount after one publish to one subscriber = 0, want non-zero")
	}
	if got := reg.SubscriberCount(); got != 1 {
		t.Fatalf("SubscriberCount after Publish = %d, want 1 (publishing must never touch the subscriber count)", got)
	}

	unsub()
	if got := reg.SendCount(); got != sendsAfterOnePublish {
		t.Fatalf("SendCount after unsubscribe = %d, want unchanged at %d", got, sendsAfterOnePublish)
	}
}

// TestLiveRegistryNoGoroutineLeak proves a publisher started and stopped
// with N subscribers attached leaks no goroutine — scoped to THIS test
// via goleak.IgnoreCurrent() rather than a package-wide TestMain, because
// this package's existing test suite opens many real Pebble stores whose
// vfs.diskHealthCheckingFS ticker goroutines wind down on their own
// schedule (verified this session: a package-wide goleak.VerifyTestMain
// fails on ~40 pre-existing, unrelated goroutines from OTHER tests in
// this package). IgnoreCurrent() snapshots whatever is already running
// (including any lingering ticker from an earlier test) and asserts only
// that nothing NEW survives past this test's own Stop — the guarantee
// the acceptance criterion actually asks for.
func TestLiveRegistryNoGoroutineLeak(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	reg := newLiveRegistry()
	var unsubs []func()
	for i := 0; i < 5; i++ {
		_, unsub := reg.Subscribe(context.Background())
		unsubs = append(unsubs, unsub)
	}

	reg.Publish(&uiv1.WatchGraphEvent{Generation: 1})
	reg.Stop()
	for _, unsub := range unsubs {
		unsub()
	}
}

// --- Task 3: the store watcher -------------------------------------------

// newTestLivePublisher starts a livePublisher over dir with a short
// debounce window (fast tests) and registers t.Cleanup to Stop it.
func newTestLivePublisher(t *testing.T, dir string) *livePublisher {
	t.Helper()
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")
	ctx, cancel := context.WithCancel(context.Background())
	pub, err := newLivePublisher(ctx, dir)
	if err != nil {
		cancel()
		t.Fatalf("newLivePublisher: %v", err)
	}
	t.Cleanup(func() {
		pub.Stop()
		cancel()
	})
	return pub
}

// TestLiveWatcherRealIndexWriteReachesSubscriber drives a REAL temporary
// index end to end: it writes a real index metadata record via the real
// store, waits past the debounce window, and asserts the subscriber
// observed the SEEDED current state first and then exactly one further
// event whose generation is strictly greater. A stubbed detector would
// not exercise fsnotify at all.
func TestLiveWatcherRealIndexWriteReachesSubscriber(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	pub := newTestLivePublisher(t, dir)
	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()

	seed, ok := recvWithTimeout(t, ch, time.Second)
	if !ok {
		t.Fatal("seed: ok = false, want true")
	}
	if seed.GetGeneration() != 1 {
		t.Fatalf("seed generation = %d, want 1", seed.GetGeneration())
	}
	if !seed.GetInitialized() {
		t.Fatal("seed against an already-indexed store: initialized = false, want true")
	}

	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})

	ev, ok := recvWithTimeout(t, ch, 5*time.Second)
	if !ok {
		t.Fatal("event after a real index write: ok = false, want true")
	}
	if ev.GetGeneration() <= seed.GetGeneration() {
		t.Fatalf("event generation = %d, want strictly greater than seed generation %d", ev.GetGeneration(), seed.GetGeneration())
	}
}

// TestLiveWatcherUnrelatedChurnProducesNoEvent proves the debounce +
// D-01 correctness filter together: touching an unrelated file (no Meta
// change) produces ZERO further events within twice the debounce window,
// paired in the SAME test with a subsequent real metadata write that
// DOES produce one — a zero-count with no positive control would prove
// only that nothing is running at all.
func TestLiveWatcherUnrelatedChurnProducesNoEvent(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	pub := newTestLivePublisher(t, dir)
	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()

	if _, ok := recvWithTimeout(t, ch, time.Second); !ok {
		t.Fatal("seed: ok = false, want true")
	}

	// Touch a file in the WATCHED parent (.codegraph/) that has nothing
	// to do with the store's own Meta record.
	unrelated := filepath.Join(dir, ".codegraph", "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("churn"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}

	assertNothingReceived(t, ch, 2*debounceMSForTest())

	// Positive control: a REAL metadata change still produces an event.
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})
	if _, ok := recvWithTimeout(t, ch, 5*time.Second); !ok {
		t.Fatal("event after a real index write following unrelated churn: ok = false, want true (positive control failed)")
	}
}

// debounceMSForTest returns the debounce window newTestLivePublisher
// configures (CODEGRAPH_DEBOUNCE_MS=20), as a time.Duration, so tests
// that need to wait "twice the debounce window" stay in lockstep with
// whatever that constant is set to.
func debounceMSForTest() time.Duration {
	return 20 * time.Millisecond
}

// TestLiveWatcherGoesLiveOnFirstIndex starts the publisher against a
// temporary repository with NO .codegraph directory at all, subscribes,
// THEN creates a real first index in that directory, and asserts the
// subscriber receives an event whose initialized is true within a
// bounded wait. This is the fix for a real defect: a non-recursive
// fsnotify watch only sees changes to entries of a directory it is
// ALREADY watching, so "start idle" alone would mean the store's later
// creation could never produce an event — a UI opened before the first
// index would be permanently deaf.
func TestLiveWatcherGoesLiveOnFirstIndex(t *testing.T) {
	dir := t.TempDir() // deliberately NO .codegraph directory at all

	pub := newTestLivePublisher(t, dir)
	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()

	seed, ok := recvWithTimeout(t, ch, time.Second)
	if !ok {
		t.Fatal("seed against an un-indexed repo: ok = false, want true")
	}
	if seed.GetInitialized() {
		t.Fatal("seed against an un-indexed repo: initialized = true, want false")
	}

	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	ev, ok := recvWithTimeout(t, ch, 5*time.Second)
	if !ok {
		t.Fatal("event after the first index was created: ok = false, want true (the publisher went live)")
	}
	if !ev.GetInitialized() {
		t.Fatal("event after the first index was created: initialized = false, want true")
	}

	// Secondary check only, per this test's own doc comment: Stop must
	// still return cleanly, but that is not the property this test
	// exists to prove.
	pub.Stop()
}

// TestLiveWatcherReArmsAfterStoreDirectoryReplacement proves T-06-39's
// re-arm half: with a live, armed publisher, RENAME the store directory
// away and move a rebuilt one into place (the shape an index rebuild or
// recovery path takes), then write real index metadata and assert an
// event still reaches the subscriber. Paired positive inside the same
// test: an event was received BEFORE the rename too, so "it re-armed"
// cannot pass on a publisher that was never armed to begin with.
func TestLiveWatcherReArmsAfterStoreDirectoryReplacement(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	pub := newTestLivePublisher(t, dir)
	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()

	seed, ok := recvWithTimeout(t, ch, time.Second)
	if !ok || seed.GetGeneration() != 1 {
		t.Fatalf("seed: got (%+v, %v), want generation 1, ok true", seed, ok)
	}

	// Positive control: a real change reaches the subscriber BEFORE the
	// rename.
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})
	preRename, ok := recvWithTimeout(t, ch, 5*time.Second)
	if !ok {
		t.Fatal("pre-rename event: ok = false, want true (positive control failed)")
	}

	storeDir := filepath.Join(dir, ".codegraph", "store")
	oldDir := filepath.Join(dir, ".codegraph", "store-old")
	if err := os.Rename(storeDir, oldDir); err != nil {
		t.Fatalf("rename store dir away: %v", err)
	}

	// Move a rebuilt store into place at the SAME path — the shape an
	// index rebuild takes: the old directory is gone, a fresh one exists
	// at the path the publisher used to watch directly.
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 3000})

	postRename, ok := recvWithTimeout(t, ch, 5*time.Second)
	if !ok {
		t.Fatal("post-rename event: ok = false, want true (the publisher went permanently deaf after the store directory was replaced)")
	}
	if postRename.GetGeneration() <= preRename.GetGeneration() {
		t.Fatalf("post-rename generation = %d, want strictly greater than pre-rename generation %d", postRename.GetGeneration(), preRename.GetGeneration())
	}
}

// TestLiveWatcherStopJoinsCleanly proves the constructor's documented
// Stop contract: it returns promptly (never hangs) and every subscriber
// channel is closed afterward.
func TestLiveWatcherStopJoinsCleanly(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pub, err := newLivePublisher(ctx, dir)
	if err != nil {
		t.Fatalf("newLivePublisher: %v", err)
	}

	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()
	if _, ok := recvWithTimeout(t, ch, time.Second); !ok {
		t.Fatal("seed: ok = false, want true")
	}

	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		pub.Stop()
	}()
	select {
	case <-stopDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return within 5s")
	}

	if ev, ok := recvWithTimeout(t, ch, time.Second); ok {
		t.Fatalf("receive after Stop: ok = true (event %+v), want false (clean close)", ev)
	}
}

// TestLiveWatcherDebounceCoalescesBurst proves the debounce window
// coalesces a burst of raw filesystem events (many Pebble-internal file
// writes across a single writeLiveMeta call, plus a following burst of
// unrelated churn) into a bounded number of published events rather than
// one per raw fsnotify event.
func TestLiveWatcherDebounceCoalescesBurst(t *testing.T) {
	dir := t.TempDir()
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 1000})

	pub := newTestLivePublisher(t, dir)
	ch, unsub := pub.Subscribe(context.Background())
	defer unsub()

	if _, ok := recvWithTimeout(t, ch, time.Second); !ok {
		t.Fatal("seed: ok = false, want true")
	}

	// A burst of unrelated writes into the watched parent directory,
	// all within one debounce window, followed by the real change.
	for i := 0; i < 5; i++ {
		p := filepath.Join(dir, ".codegraph", fmt.Sprintf("churn-%d.txt", i))
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write churn file %d: %v", i, err)
		}
	}
	writeLiveMeta(t, dir, &schema.Meta{SchemaVersion: schema.SchemaVersion, LastSyncUnixMs: 2000})

	ev, ok := recvWithTimeout(t, ch, 5*time.Second)
	if !ok {
		t.Fatal("event after the burst: ok = false, want true")
	}
	if ev.GetGeneration() != 2 {
		t.Fatalf("event generation after one real change amid a burst = %d, want 2 (exactly one published event, not one per raw fsnotify event)", ev.GetGeneration())
	}

	assertNothingReceived(t, ch, 2*debounceMSForTest())
}
