package uiserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/goleak"

	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// --- runWatchGraphLoop classification tests (no HTTP required) ---------

// fakeWatchGraphSender is a watchGraphSender test double: it always
// returns sendErr from Send, letting the classification tests drive a
// send failure deterministically without a real connect.ServerStream,
// which has no exported constructor.
type fakeWatchGraphSender struct {
	sendErr error
}

func (f *fakeWatchGraphSender) Send(*uiv1.WatchGraphEvent) error {
	return f.sendErr
}

// TestWatchGraphHandlerDisconnectClassifiedAsCancellation asserts one
// half of the classification branch: an already-cancelled request
// context (a closed tab) produces a CodeCanceled error, even before any
// event or send is involved.
func TestWatchGraphHandlerDisconnectClassifiedAsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := make(chan *uiv1.WatchGraphEvent)
	err := runWatchGraphLoop(ctx, ch, &fakeWatchGraphSender{})
	if err == nil {
		t.Fatal("runWatchGraphLoop with an already-cancelled context returned nil, want a cancellation error")
	}
	if code := connect.CodeOf(err); code != connect.CodeCanceled {
		t.Fatalf("code = %v, want CodeCanceled for a cancelled client context", code)
	}
}

// TestWatchGraphHandlerSendFailureWithLiveContextIsNotClassifiedAsCancellation
// asserts the OTHER half: a send failure raised while the context is
// still live must NOT be reported as a cancellation — a handler that
// hard-codes cancellation for every failure fails this test even though
// it passes the previous one.
func TestWatchGraphHandlerSendFailureWithLiveContextIsNotClassifiedAsCancellation(t *testing.T) {
	ctx := context.Background() // never cancelled

	ch := make(chan *uiv1.WatchGraphEvent, 1)
	ch <- &uiv1.WatchGraphEvent{Generation: 1}

	wantErr := errors.New("broken pipe")
	err := runWatchGraphLoop(ctx, ch, &fakeWatchGraphSender{sendErr: wantErr})
	if err == nil {
		t.Fatal("runWatchGraphLoop with a send failure returned nil, want an error")
	}
	if code := connect.CodeOf(err); code == connect.CodeCanceled {
		t.Fatal("code = CodeCanceled for a send failure raised with a LIVE context — a genuine transport failure was mislabelled as a client cancellation")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want it to wrap %v", err, wantErr)
	}
}

// TestWatchGraphHandlerPublisherChannelCloseEndsLoopCleanly asserts the
// third runWatchGraphLoop branch directly: the channel closing (the
// publisher stopping) ends the loop with a nil error, never a
// cancellation and never a transport error.
func TestWatchGraphHandlerPublisherChannelCloseEndsLoopCleanly(t *testing.T) {
	ctx := context.Background()
	ch := make(chan *uiv1.WatchGraphEvent)
	close(ch)

	if err := runWatchGraphLoop(ctx, ch, &fakeWatchGraphSender{}); err != nil {
		t.Fatalf("runWatchGraphLoop with a closed channel returned %v, want nil (a clean stream end)", err)
	}
}

// --- Real HTTP/1.1, real generated client -------------------------------

// waitForSubscriberCount polls srv's publisher registry until its
// subscriber count equals want or a bounded deadline elapses.
func waitForSubscriberCount(t *testing.T, srv *Server, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := srv.publisher.registry.SubscriberCount(); got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("subscriber count never reached %d (last observed %d)", want, srv.publisher.registry.SubscriberCount())
}

// TestWatchGraphHandlerRegistersAndDeregistersSubscriber proves the
// subscriber count goes UP by exactly one while a real stream is open
// and back DOWN to its prior value once it closes — asserting the rise
// as well as the fall, so a handler that never registered at all cannot
// pass by satisfying only the upper bound.
func TestWatchGraphHandlerRegistersAndDeregistersSubscriber(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	before := srv.publisher.registry.SubscriberCount()

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	streamCtx, cancelStream := context.WithCancel(context.Background())
	stream, err := client.WatchGraph(streamCtx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive (seeded message): %v", stream.Err())
	}

	waitForSubscriberCount(t, srv, before+1)

	cancelStream()
	_ = stream.Close()

	waitForSubscriberCount(t, srv, before)
}

// TestWatchGraphHandlerFirstMessageIsCurrentState proves a freshly
// opened stream delivers the publisher's CURRENT state as its first
// message, before any change occurs — paired on generation AND on the
// other fields, so "it received a message" cannot be satisfied by a
// zero-valued event that merely happens to share a generation number.
func TestWatchGraphHandlerFirstMessageIsCurrentState(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	want := srv.publisher.registry.Current()
	if want == nil {
		t.Fatal("publisher's current event is nil immediately after Listen — newLivePublisher must run its bootstrap check synchronously before returning")
	}

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	stream, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
	got := stream.Msg()
	if got.GetGeneration() != want.GetGeneration() {
		t.Fatalf("first message generation = %d, want %d (the publisher's current state at Subscribe time)", got.GetGeneration(), want.GetGeneration())
	}
	if got.GetInitialized() != want.GetInitialized() || got.GetStoreExists() != want.GetStoreExists() {
		t.Fatalf("first message = %+v, want it to equal the publisher's current state %+v", got, want)
	}
}

// TestWatchGraphHandlerDeliversEventsInOrder publishes three events one
// at a time, receiving each before publishing the next, and asserts they
// arrive in the exact generation order published — basic sanity ahead of
// the coalescing test below, which deliberately does NOT read between
// publishes.
func TestWatchGraphHandlerDeliversEventsInOrder(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	stream, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() { // the seeded current-state message
		t.Fatalf("Receive (seeded): %v", stream.Err())
	}

	for i := int64(2); i <= 4; i++ {
		srv.publisher.registry.Publish(&uiv1.WatchGraphEvent{Generation: i})
		if !stream.Receive() {
			t.Fatalf("Receive (generation %d): %v", i, stream.Err())
		}
		if got := stream.Msg().GetGeneration(); got != i {
			t.Fatalf("received generation %d, want %d — events out of order", got, i)
		}
	}
}

// coalescePublishCount is the number of generations
// TestWatchGraphHandlerCoalescesForANonReadingClient publishes without
// the client ever reading — 5000 x 65 bytes is ~325 KB, deliberately
// UNDER the ~496 KB blocking threshold measured on this project's own
// development platform (net.inet.tcp.sendspace/recvspace = 131072,
// ~7,825 messages before a raw Write blocks). The gate does not rely on
// the socket ever blocking: the capacity-1 registry discards whenever
// the publisher outruns the send loop, which a paused reader guarantees
// regardless of platform buffer sizes. Do not raise this count to try to
// force a socket stall — that repeats a measured, disproven approach
// (see the SUMMARY).
const coalescePublishCount = 5000

// TestWatchGraphHandlerCoalescesForANonReadingClient is the one place
// D-04's backpressure is observed over a real socket: a real stream is
// opened with the real generated client over httptest HTTP/1.1, and the
// test does NOT call Receive at all while publishing. Only after every
// publish has been issued does it resume receiving, and the four
// assertions below hold together: at most half of what was published
// arrived (a ratio, not merely "fewer"), at least one message arrived,
// strictly fewer arrived than were published, and the LAST one received
// carries the LAST generation published — proving coalescing, not
// dropping.
func TestWatchGraphHandlerCoalescesForANonReadingClient(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	stream, err := client.WatchGraph(streamCtx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	defer stream.Close()

	waitForSubscriberCount(t, srv, 1)

	// Publish as fast as possible with NO Receive call in between — the
	// registry never blocks (D-04), so this loop completes essentially
	// instantly regardless of whether the client is reading.
	var lastPublished int64
	for i := 0; i < coalescePublishCount; i++ {
		gen := int64(i + 2) // +2: generation 1 is the seeded current state
		srv.publisher.registry.Publish(&uiv1.WatchGraphEvent{Generation: gen})
		lastPublished = gen
	}

	type result struct {
		ev *uiv1.WatchGraphEvent
		ok bool
	}
	resultCh := make(chan result, 1)
	go func() {
		for {
			ok := stream.Receive()
			r := result{ok: ok}
			if ok {
				r.ev = stream.Msg()
			}
			resultCh <- r
			if !ok {
				return
			}
		}
	}()

	var received []*uiv1.WatchGraphEvent
	idle := time.NewTimer(500 * time.Millisecond)
	defer idle.Stop()
drain:
	for {
		select {
		case r := <-resultCh:
			if !r.ok {
				break drain
			}
			// Copy the message: stream.Msg() is only valid until the
			// next Receive call.
			received = append(received, &uiv1.WatchGraphEvent{
				Generation: r.ev.GetGeneration(),
			})
			if !idle.Stop() {
				<-idle.C
			}
			idle.Reset(500 * time.Millisecond)
		case <-idle.C:
			cancelStream()
			break drain
		}
	}

	published := coalescePublishCount
	got := len(received)

	if got < 1 {
		t.Fatal("received 0 messages, want at least 1 — fewer must not mean none")
	}
	if got >= published {
		t.Fatalf("received %d, want strictly fewer than published %d", got, published)
	}
	if got*2 > published {
		t.Fatalf("received %d of %d published (ratio %.4f), want received*2 <= published — a coalescing RATIO, not merely fewer", got, published, float64(got)/float64(published))
	}
	if last := received[len(received)-1].GetGeneration(); last != lastPublished {
		t.Fatalf("last received generation = %d, want %d (the newest published value) — coalescing must keep the newest, not merely discard older ones", last, lastPublished)
	}

	t.Logf("coalescing observed: published=%d received=%d ratio=%.4f last_generation=%d", published, got, float64(got)/float64(published), received[len(received)-1].GetGeneration())
}

// TestWatchGraphHandlerMultipleSubscribersIndependentChannels proves the
// fan-out registry delivers the same event to two independently-open
// streams, each on its own channel.
func TestWatchGraphHandlerMultipleSubscribersIndependentChannels(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	streamA, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph A: %v", err)
	}
	defer streamA.Close()
	streamB, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph B: %v", err)
	}
	defer streamB.Close()

	if !streamA.Receive() {
		t.Fatalf("Receive A (seeded): %v", streamA.Err())
	}
	if !streamB.Receive() {
		t.Fatalf("Receive B (seeded): %v", streamB.Err())
	}

	waitForSubscriberCount(t, srv, 2)

	srv.publisher.registry.Publish(&uiv1.WatchGraphEvent{Generation: 42})

	if !streamA.Receive() {
		t.Fatalf("Receive A: %v", streamA.Err())
	}
	if got := streamA.Msg().GetGeneration(); got != 42 {
		t.Fatalf("stream A generation = %d, want 42", got)
	}
	if !streamB.Receive() {
		t.Fatalf("Receive B: %v", streamB.Err())
	}
	if got := streamB.Msg().GetGeneration(); got != 42 {
		t.Fatalf("stream B generation = %d, want 42", got)
	}
}

// TestWatchGraphHandlerResumeCursorGreaterThanZeroDoesNotError proves a
// since_generation greater than zero does not error: the server holds no
// history to replay, and the seeded current-state message is what makes
// that safe rather than lossy.
func TestWatchGraphHandlerResumeCursorGreaterThanZeroDoesNotError(t *testing.T) {
	dir := t.TempDir()
	srv := startedServer(t, dir)

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	stream, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{SinceGeneration: 999}))
	if err != nil {
		t.Fatalf("WatchGraph with since_generation=999: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
}

// TestWatchGraphHandlerPublisherStopEndsStreamCleanly proves the
// publisher stopping (server shutdown) closes the subscriber channel and
// the handler returns cleanly, WITHOUT an error — the client sees a
// normal stream end and its own backoff handles the reconnect.
func TestWatchGraphHandlerPublisherStopEndsStreamCleanly(t *testing.T) {
	dir := t.TempDir()
	srv := mustListen(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	stream, err := client.WatchGraph(context.Background(), connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("Receive (seeded): %v", stream.Err())
	}

	srv.stopPublisher()

	if stream.Receive() {
		t.Fatal("Receive after the publisher stopped succeeded, want the stream to end")
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream ended with err = %v, want nil (a clean stream end, not an error)", err)
	}

	cancel()
	<-serveErrCh
	_ = srv.Close()
}

// TestWatchGraphHandlerNoGoroutineLeak proves N stream open/close cycles
// leave no goroutine behind. Server lifecycle is managed with explicit
// defers (not t.Cleanup) so cleanup runs BEFORE the deferred goleak
// check below, per Go's own ordering: defers registered later run
// first, so cancel()/Close() (deferred after the goleak check) unwind
// before goleak.VerifyNone inspects what remains.
func TestWatchGraphHandlerNoGoroutineLeak(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	dir := t.TempDir()
	srv := mustListen(t, dir)
	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	for i := 0; i < 3; i++ {
		streamCtx, cancelStream := context.WithCancel(context.Background())
		stream, err := client.WatchGraph(streamCtx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
		if err != nil {
			t.Fatalf("WatchGraph iteration %d: %v", i, err)
		}
		if !stream.Receive() {
			t.Fatalf("Receive (seeded) iteration %d: %v", i, stream.Err())
		}
		cancelStream()
		_ = stream.Close()
		waitForSubscriberCount(t, srv, 0)
	}

	// Wait for a fully graceful shutdown to complete BEFORE the deferred
	// goleak check runs, rather than relying on defer-ordering tricks —
	// goleak.VerifyNone is the FIRST defer registered, so it is the
	// LAST to run, but only once this function's body actually returns.
	cancel()
	select {
	case <-serveErrCh:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after cancel")
	}
	_ = srv.Close()
}

// --- Server publisher lifetime -------------------------------------------

// TestServerPublisherLifetimeReleasedOnCloseWithoutServe proves Listen
// followed directly by Close — Serve never called — leaks no fsnotify
// watcher or debouncer goroutine.
func TestServerPublisherLifetimeReleasedOnCloseWithoutServe(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	srv := mustListen(t, t.TempDir())
	if err := srv.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestServerPublisherLifetimeShutdownIsPromptWithAnOpenStream proves
// cancelling the context passed to Serve, with a stream open, returns
// from Serve in well under the 5-second Shutdown budget (T-06-43).
// Without stopping the publisher BEFORE Shutdown, the open stream is an
// in-flight connection that would hold Shutdown until its full budget
// expired.
func TestServerPublisherLifetimeShutdownIsPromptWithAnOpenStream(t *testing.T) {
	dir := t.TempDir()
	srv := mustListen(t, dir)
	ctx, cancel := context.WithCancel(context.Background())

	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))

	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())
	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	stream, err := client.WatchGraph(streamCtx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	if !stream.Receive() {
		t.Fatalf("Receive (seeded): %v", stream.Err())
	}

	start := time.Now()
	cancel() // simulates Ctrl-C: cancel the SERVER's own Serve context
	select {
	case err := <-serveErrCh:
		if err != nil {
			t.Fatalf("Serve returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return within 2s of context cancellation with an open stream, want well under the 5s Shutdown budget")
	}
	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Fatalf("Serve took %v to return, want under 2s", elapsed)
	}
	_ = srv.Close()
}

// TestServerPublisherLifetimeStopIsIdempotent proves calling Close twice
// does not panic — Serve's ctx.Done() branch and Close both call
// stopPublisher, and a caller may reasonably do both. The underlying
// net.Listener itself is not idempotent to close twice (its second
// Close legitimately returns "use of closed network connection"), so
// only the FIRST call's error is asserted nil; the property under test
// is that stopPublisher's own sync.Once prevents a double-Stop panic on
// the SECOND call, whatever the listener itself reports.
func TestServerPublisherLifetimeStopIsIdempotent(t *testing.T) {
	srv := mustListen(t, t.TempDir())
	if err := srv.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	_ = srv.Close() // must not panic
}

// --- Task 3: the phase's tracer — end to end, real client, real store ---

// TestWatchGraphStreamDeliversMessageByMessage is the phase's tracer
// assertion for criterion 2: it wires a real index write on disk,
// through the fsnotify watcher, the debouncer, the change detector, the
// publisher's registry, the deadline-cleared middleware, real HTTP/1.1,
// and into a real generated Connect client — with no stub anywhere on
// the path.
//
// It asserts TIMING, not arrival. A test that merely counts N received
// messages is vacuous here: silent buffering delivers all N at the very
// end of the stream and would still pass an "it all arrived" check. This
// test is structured so buffering FAILS instead: it blocks on RECEIVING
// message k, over the real socket, before triggering the write for
// message k+1. A handler that accumulates events and only flushes them
// when the stream ends can never satisfy that blocking Receive before
// the next trigger fires, so it times out — this ordering IS the
// assertion, not a timestamp comparison performed after the fact, which
// a buffered implementation could still satisfy by accident.
//
// RED was demonstrated this session: with the send loop temporarily
// changed to accumulate every event and flush them all only when the
// stream ends (the exact buffering failure this test exists to catch),
// this test FAILED — it hung on Receive (blocked past the stream's own
// 5-second deadline) discarding the seeded message, because a buffering
// handler holds EVERY event, including the seed, until the stream ends.
// See the SUMMARY for the captured failure output. Reverting the send
// loop to send-as-you-go made it pass again.
func TestWatchGraphStreamDeliversMessageByMessage(t *testing.T) {
	t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")

	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	// A generous overall deadline bounds the whole stream: a correct,
	// incremental implementation finishes in well under a second (three
	// triggers, 150ms apart), while a buffering implementation blocks
	// until this deadline and the test fails on it rather than hanging
	// the suite indefinitely.
	streamCtx, cancelStream := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelStream()

	stream, err := client.WatchGraph(streamCtx, connect.NewRequest(&uiv1.WatchGraphRequest{}))
	if err != nil {
		t.Fatalf("WatchGraph: %v", err)
	}
	defer stream.Close()

	// Step 0: receive and DISCARD the seeded first message. Every
	// stream's first message is the publisher's current state (06-02's
	// Subscribe contract), which arrives at connect time and is
	// therefore not a triggered event — consuming it explicitly keeps
	// the trigger accounting below exact rather than off by one.
	if !stream.Receive() {
		t.Fatalf("Receive (seeded message): %v", stream.Err())
	}
	t.Logf("discarded seeded message: generation=%d", stream.Msg().GetGeneration())

	const triggerCount = 3
	// triggerSpacing must clear the 20ms debounce window with generous
	// margin so a slow CI machine's write+debounce+publish latency
	// cannot itself produce a false inter-arrival-gap failure.
	const triggerSpacing = 150 * time.Millisecond

	baseMs := time.Now().UnixMilli()
	receiptTimes := make([]time.Time, 0, triggerCount)
	receivedGenerations := make([]int64, 0, triggerCount)

	for i := 0; i < triggerCount; i++ {
		if i > 0 {
			time.Sleep(triggerSpacing)
		}

		// Step 1: trigger a REAL index metadata change — a real store
		// write through graphstore, not a synthetic
		// registry.Publish call. This is what forces the write to
		// travel through fsnotify and the debouncer for real.
		writeLiveMeta(t, dir, &schema.Meta{
			SchemaVersion:  1,
			LastSyncUnixMs: baseMs + int64(i+1)*1000,
			Healthy:        true,
		})

		// Steps 2-3: BLOCK until the client has received this exact
		// message before the loop proceeds to trigger the next one.
		// This blocking-before-next-trigger structure is the timing
		// assertion itself.
		if !stream.Receive() {
			t.Fatalf("Receive (trigger %d): %v", i, stream.Err())
		}
		receiptTimes = append(receiptTimes, time.Now())
		receivedGenerations = append(receivedGenerations, stream.Msg().GetGeneration())
	}

	// Step 4 repeated triggerCount times above. Now verify the
	// accounting: the triggered-receipt count must equal the trigger
	// count (paired with the interval bound below — an inter-arrival
	// minimum computed over a single receipt would be vacuous), and
	// every consecutive gap between triggered receipts must be at least
	// the deliberate trigger spacing.
	if got := len(receivedGenerations); got != triggerCount {
		t.Fatalf("received %d triggered messages, want exactly %d", got, triggerCount)
	}

	for i := 1; i < len(receiptTimes); i++ {
		gap := receiptTimes[i].Sub(receiptTimes[i-1])
		if gap < triggerSpacing {
			t.Fatalf("inter-arrival gap[%d] = %v, want at least the trigger spacing %v — message %d arrived before message %d's trigger could plausibly have caused it", i, gap, triggerSpacing, i, i-1)
		}
	}

	t.Logf("triggered receipts: generations=%v, inter-arrival gaps all >= trigger spacing %v", receivedGenerations, triggerSpacing)
}
