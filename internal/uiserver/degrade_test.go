// Package-level scope note for this file (SRV-04's degrade behavior
// tests): the exact FINAL-ATTEMPT boundary of graphstore.Open's bounded
// retry budget is owned by, and tested in, internal/graphstore —
// TestOpenSucceedsOnTheFinalAttempt (internal/graphstore/open_lock_test.go)
// — because the openLockRetrySleep seam it uses is unexported and
// unreachable from this package. This file deliberately covers only the
// two BEHAVIORAL sides of that boundary: TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget
// (a causally-released holder, which MUST succeed) and
// TestNonStatusRPCsDegradeWhenAHolderNeverReleases (a never-released
// holder, which MUST degrade). Neither test here reaches for
// openLockRetrySleep or reimplements a retry-aware clock.
//
// Deferred build-tagged integration coverage discovered while exercising
// this degrade path against real processes (Task 3) is recorded in
// 01-11-SUMMARY.md under "## Deferred: build-tagged integration
// coverage".
package uiserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// TestClassifyDegradePrecedence drives classifyDegrade directly with
// constructed errors rather than attempting to build an impossible
// repository on disk: a test that cannot reach its own precondition
// proves nothing. not-initialized outranks locked (classifyDegrade's own
// doc comment records why), asserted here with a synthetic error
// satisfying BOTH sentinels via fmt.Errorf's multi-%w wrapping.
func TestClassifyDegradePrecedence(t *testing.T) {
	t.Run("not-initialized", func(t *testing.T) {
		if got := classifyDegrade(query.ErrNotInitialized); got != degradeNotInitialized {
			t.Fatalf("classifyDegrade(query.ErrNotInitialized) = %v, want degradeNotInitialized", got)
		}
	})

	t.Run("store-locked", func(t *testing.T) {
		if got := classifyDegrade(graphstore.ErrStoreLocked); got != degradeIndexingInProgress {
			t.Fatalf("classifyDegrade(graphstore.ErrStoreLocked) = %v, want degradeIndexingInProgress", got)
		}
	})

	t.Run("nil-error", func(t *testing.T) {
		if got := classifyDegrade(nil); got != degradeNone {
			t.Fatalf("classifyDegrade(nil) = %v, want degradeNone", got)
		}
	})

	t.Run("unrelated-error", func(t *testing.T) {
		if got := classifyDegrade(errors.New("some unrelated failure")); got != degradeNone {
			t.Fatalf("classifyDegrade(unrelated error) = %v, want degradeNone", got)
		}
	})

	t.Run("satisfies-both-sentinels", func(t *testing.T) {
		// A repository cannot genuinely be both uninitialized and
		// lock-contended on a real filesystem (classifyDegrade's own doc
		// comment) — this constructed error exists only to prove the
		// classifier's precedence is an ORDERED CHECK rather than an
		// accident of evaluation order, on the one input that can
		// actually exercise it: a synthetic error, not a fabricated
		// repository.
		both := fmt.Errorf("synthetic dual-sentinel error: %w and also %w", query.ErrNotInitialized, graphstore.ErrStoreLocked)
		if !errors.Is(both, query.ErrNotInitialized) || !errors.Is(both, graphstore.ErrStoreLocked) {
			t.Fatal("test construction broken: synthetic error does not satisfy both sentinels")
		}
		if got := classifyDegrade(both); got != degradeNotInitialized {
			t.Fatalf("classifyDegrade(error satisfying both sentinels) = %v, want degradeNotInitialized (not-initialized must outrank locked)", got)
		}
	})
}

// TestStatusDegradesOnLock proves D-16: with the store held past Open's
// bounded retry budget by a REAL graphstore.Open handle in this test
// process (not a mocked or injected error — a second goroutine's handle
// reproduces the same Pebble directory lock the daemon takes),
// GetStatus returns a SUCCESSFUL response — never an error — reporting
// that the store exists and that indexing is in progress, with
// graph-derived counts zeroed.
func TestStatusDegradesOnLock(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}
	t.Cleanup(func() { holder.Close() })

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus while the store is locked past the retry budget returned an error, want a degraded SUCCESSFUL response: %v", err)
	}
	if resp.Msg.GetInitialized() {
		t.Fatal("GetStatus while locked: initialized = true, want false (the Engine was never successfully opened)")
	}
	if !resp.Msg.GetStoreExists() {
		t.Fatal("GetStatus while locked: store_exists = false, want true (a .codegraph/ directory was found)")
	}
	if !resp.Msg.GetIndexingInProgress() {
		t.Fatal("GetStatus while locked: indexing_in_progress = false, want true")
	}
	if resp.Msg.GetNodeCount() != 0 || resp.Msg.GetEdgeCount() != 0 || resp.Msg.GetFileCount() != 0 {
		t.Fatalf("GetStatus while locked: graph-derived counts not zeroed (node=%d edge=%d file=%d)", resp.Msg.GetNodeCount(), resp.Msg.GetEdgeCount(), resp.Msg.GetFileCount())
	}
}

// TestStatusOnUninitializedRepo proves the not-initialized degrade state
// is distinct from the locked state: a directory with no .codegraph/ at
// all returns a successful response with BOTH the initialized flag and
// the indexing flag unset.
func TestStatusOnUninitializedRepo(t *testing.T) {
	dir := t.TempDir() // never indexed — no .codegraph/ directory exists

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus on an uninitialized repository returned an error, want a degraded SUCCESSFUL response: %v", err)
	}
	if resp.Msg.GetInitialized() {
		t.Fatal("GetStatus on an uninitialized repository: initialized = true, want false")
	}
	if resp.Msg.GetIndexingInProgress() {
		t.Fatal("GetStatus on an uninitialized repository: indexing_in_progress = true, want false — the two degrade states must be distinct")
	}
	if resp.Msg.GetStoreExists() {
		t.Fatal("GetStatus on an uninitialized repository: store_exists = true, want false — no .codegraph/ directory exists")
	}
}

// containsPathSeparator reports whether s contains either OS-style path
// separator. Test-only helper for TestIndexingInProgressMessageLeaksNothing,
// positive-controlled in the same test by confirming it reports true for
// a string that does contain one.
func containsPathSeparator(s string) bool {
	return strings.ContainsAny(s, "/\\")
}

// TestIndexingInProgressMessageLeaksNothing proves T-01-06's mitigation:
// the message equals the named constant exactly, and the constant
// contains no path separator of either kind — the helper doing that
// check is positive-controlled by confirming it reports true for a
// string that does contain one, in this same test.
func TestIndexingInProgressMessageLeaksNothing(t *testing.T) {
	if indexingInProgressMessage == "" {
		t.Fatal("indexingInProgressMessage is empty, want a fixed non-empty generic constant")
	}
	if indexingInProgressMessage != "The index is being rebuilt. Please retry shortly." {
		t.Fatalf("indexingInProgressMessage = %q, want the exact named constant", indexingInProgressMessage)
	}
	if containsPathSeparator(indexingInProgressMessage) {
		t.Fatalf("indexingInProgressMessage %q contains a path separator, want none", indexingInProgressMessage)
	}
	// Positive control: the helper must actually detect a separator when
	// one is present, or the negative check above proves nothing.
	if !containsPathSeparator("/var/tmp/example/.codegraph/store/LOCK") {
		t.Fatal("positive control failed: containsPathSeparator did not detect a path separator in a string that has one")
	}
	if !containsPathSeparator(`C:\Users\example\.codegraph\store\LOCK`) {
		t.Fatal("positive control failed: containsPathSeparator did not detect a backslash path separator in a string that has one")
	}
}

// namedRPCCall pairs an rpc's name with a closure invoking it, so
// TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget and
// TestNonStatusRPCsDegradeWhenAHolderNeverReleases can drive every rpc
// (or every non-GetStatus rpc) uniformly, one t.Run subtest per name.
type namedRPCCall struct {
	name string
	call func() error
}

// allNineRPCCalls returns one namedRPCCall per uiv1connect.UIServiceClient
// method, each carrying arguments that succeed against a gofixture-indexed
// repository (see copyGofixture/indexGofixture and other tests in this
// package that use the same "Alpha"/"helper"/"pkga/pkga.go" fixture
// symbols).
func allNineRPCCalls(client uiv1connect.UIServiceClient) []namedRPCCall {
	ctx := context.Background()
	return []namedRPCCall{
		{"GetStatus", func() error {
			_, err := client.GetStatus(ctx, connect.NewRequest(&uiv1.GetStatusRequest{}))
			return err
		}},
		{"Search", func() error {
			_, err := client.Search(ctx, connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha"}))
			return err
		}},
		{"Files", func() error {
			_, err := client.Files(ctx, connect.NewRequest(&uiv1.FilesRequest{Format: "flat"}))
			return err
		}},
		{"Callers", func() error {
			_, err := client.Callers(ctx, connect.NewRequest(&uiv1.CallersRequest{Symbol: "helper"}))
			return err
		}},
		{"Callees", func() error {
			_, err := client.Callees(ctx, connect.NewRequest(&uiv1.CalleesRequest{Symbol: "Alpha"}))
			return err
		}},
		{"Impact", func() error {
			_, err := client.Impact(ctx, connect.NewRequest(&uiv1.ImpactRequest{Symbol: "helper"}))
			return err
		}},
		{"Affected", func() error {
			_, err := client.Affected(ctx, connect.NewRequest(&uiv1.AffectedRequest{Files: []string{"pkga/pkga.go"}}))
			return err
		}},
		{"GetNodeDetail", func() error {
			_, err := client.GetNodeDetail(ctx, connect.NewRequest(&uiv1.GetNodeDetailRequest{File: "pkga/pkga.go"}))
			return err
		}},
		{"Explore", func() error {
			_, err := client.Explore(ctx, connect.NewRequest(&uiv1.ExploreRequest{Query: "Alpha"}))
			return err
		}},
	}
}

// nonStatusRPCCalls is allNineRPCCalls minus GetStatus — the eight rpcs
// that map a locked store to connect.CodeUnavailable rather than to
// D-16's degraded-but-successful GetStatus response.
func nonStatusRPCCalls(client uiv1connect.UIServiceClient) []namedRPCCall {
	all := allNineRPCCalls(client)
	out := make([]namedRPCCall, 0, len(all)-1)
	for _, c := range all {
		if c.name == "GetStatus" {
			continue
		}
		out = append(out, c)
	}
	return out
}

// TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget proves the
// TRANSIENT-collision case: a holder that acquires the store and
// releases it on a CAUSAL EDGE — a channel fired from the wrapped
// openEngine package var on its FIRST invocation, never a timer sized
// against graphstore.Open's budget — must let EVERY one of the nine RPCs
// return a normal successful result. A degrade here is a failure,
// because riding out a transient collision is exactly what the bounded
// retry exists for and what internal/cli/daemon.go (opens only inside
// its flush, then closes) and internal/mcp/tools.go (opens per tool
// call) actually produce: neither holds the store long-term, so a real
// `codegraph daemon` or `serve --mcp` running concurrently with
// `codegraph ui` collides only transiently.
//
// The release is event-synchronized, not timer-scheduled: wrapping
// openEngine so its first call signals a channel establishes a
// happens-before edge between "an RPC is about to open the store" and
// "the holder has begun releasing it". The only way the first RPC could
// still fail is if a single Close() call took longer than the ENTIRE
// retry budget — a real defect worth failing on, not a scheduling
// artifact. A timer-based release "comfortably inside" the ~400ms budget
// was considered and rejected: it is still wall-clock coordination and
// can cross the boundary spuriously on a loaded -race runner.
func TestRPCsSucceedWhenAHolderReleasesWithinTheOpenBudget(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}

	release := make(chan struct{})
	var once sync.Once
	orig := openEngine
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		once.Do(func() { close(release) })
		return orig(start)
	}
	t.Cleanup(func() { openEngine = orig })

	closeErr := make(chan error, 1)
	go func() {
		<-release
		closeErr <- holder.Close()
	}()

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	for _, c := range allNineRPCCalls(client) {
		t.Run(c.name, func(t *testing.T) {
			if err := c.call(); err != nil {
				t.Fatalf("%s returned an error, want a normal successful result under a transiently-released holder: %v", c.name, err)
			}
		})
	}

	if err := <-closeErr; err != nil {
		t.Fatalf("holder Close: %v", err)
	}
}

// indexingInProgressDetail extracts and unmarshals the *uiv1.IndexingInProgress
// detail from a connect error, failing the test if err is not a
// *connect.Error, carries no detail, or no detail unmarshals to that
// type. Used to assert on the STRUCTURED detail, never on the error
// string.
func indexingInProgressDetail(t *testing.T, err error) *uiv1.IndexingInProgress {
	t.Helper()
	var connErr *connect.Error
	if !errors.As(err, &connErr) {
		t.Fatalf("error %v is not a *connect.Error", err)
	}
	for _, d := range connErr.Details() {
		msg, valueErr := d.Value()
		if valueErr != nil {
			continue
		}
		if iip, ok := msg.(*uiv1.IndexingInProgress); ok {
			return iip
		}
	}
	t.Fatalf("error %v carries no IndexingInProgress detail", err)
	return nil
}

// TestNonStatusRPCsDegradeWhenAHolderNeverReleases proves the SUSTAINED
// case — criterion 3's second clause, "a re-index that outlasts
// graphstore.Open's retry budget renders as indexing in progress": with
// the store held for the whole test, EVERY one of the eight non-GetStatus
// RPCs returns exactly connect.CodeUnavailable with a detail that
// unmarshals to an IndexingInProgress. A success here is a failure, and
// so is connect.CodeInternal.
func TestNonStatusRPCsDegradeWhenAHolderNeverReleases(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}
	t.Cleanup(func() { holder.Close() })

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	for _, c := range nonStatusRPCCalls(client) {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil {
				t.Fatalf("%s succeeded while the store was locked past the retry budget, want CodeUnavailable", c.name)
			}
			if code := connect.CodeOf(err); code != connect.CodeUnavailable {
				t.Fatalf("%s: code = %v, want CodeUnavailable", c.name, code)
			}
			detail := indexingInProgressDetail(t, err)
			if detail.GetMessage() == "" {
				t.Fatalf("%s: IndexingInProgress.message is empty", c.name)
			}
		})
	}
}

// TestDegradedRPCOpensTheStoreExactlyOnce proves D-15 directly: with
// openEngine swapped for a counting wrapper and the store held for the
// duration, one degraded RPC call invokes openEngine exactly once. A
// second retry layer above graphstore.Open would have to open again, so
// the count would exceed 1. This is deterministic under parallel CI
// load, unlike a timing assertion around the ~400ms budget would be —
// no test in this file asserts on elapsed wall-clock time.
func TestDegradedRPCOpensTheStoreExactlyOnce(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}
	t.Cleanup(func() { holder.Close() })

	var opens int64
	orig := openEngine
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		atomic.AddInt64(&opens, 1)
		return orig(start)
	}
	t.Cleanup(func() { openEngine = orig })

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err = client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha"}))
	if err == nil {
		t.Fatal("Search succeeded while the store was locked past the retry budget, want a degrade")
	}
	if got := atomic.LoadInt64(&opens); got != 1 {
		t.Fatalf("openEngine invoked %d times for one degraded RPC call, want exactly 1", got)
	}
}

// snapshotRepoFiles walks dir and returns a map of relative path to
// (size, mtime) for every regular file — a cheap fingerprint used by
// TestDegradeIsIdempotent to prove a degraded call mutates nothing under
// the repository.
func snapshotRepoFiles(t *testing.T, dir string) map[string][2]int64 {
	t.Helper()
	out := make(map[string][2]int64)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		out[rel] = [2]int64{info.Size(), info.ModTime().UnixNano()}
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotRepoFiles(%s): %v", dir, err)
	}
	return out
}

// TestDegradeIsIdempotent proves calling the same RPC twice against a
// permanently locked store yields the identical connect.CodeUnavailable
// and an identical detail message both times, and mutates nothing under
// the repository between the two calls (SRV-04's idempotency edge).
func TestDegradeIsIdempotent(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}
	t.Cleanup(func() { holder.Close() })

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	before := snapshotRepoFiles(t, dir)

	_, err1 := client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha"}))
	_, err2 := client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha"}))

	after := snapshotRepoFiles(t, dir)

	if err1 == nil || err2 == nil {
		t.Fatalf("expected both calls to degrade, got err1=%v err2=%v", err1, err2)
	}
	code1, code2 := connect.CodeOf(err1), connect.CodeOf(err2)
	if code1 != connect.CodeUnavailable || code2 != connect.CodeUnavailable {
		t.Fatalf("codes = %v, %v, want both CodeUnavailable", code1, code2)
	}
	if code1 != code2 {
		t.Fatalf("codes differ across two calls under the same permanent hold: %v vs %v", code1, code2)
	}
	msg1 := indexingInProgressDetail(t, err1).GetMessage()
	msg2 := indexingInProgressDetail(t, err2).GetMessage()
	if msg1 != msg2 {
		t.Fatalf("detail messages differ across two calls under the same permanent hold: %q vs %q", msg1, msg2)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("repository files changed between two degraded calls, want no mutation")
	}
}

// TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete proves SRV-04's
// actual property — no handle retained BETWEEN calls — the behavioral
// way: N concurrent RPCs are issued and EVERY one is awaited to
// completion (each handler's withEngine, or GetStatus's own direct
// openEngine call, closes its handle before the handler itself returns —
// so by the time every client call has returned, every server-side
// handle has already been released) before this test's own
// graphstore.Open on the same store directory is attempted. By
// construction there is no contention left at that instant: this needs
// no fairness assumption about who wins a race, unlike the rejected
// TestConcurrentRPCsDoNotStarveAHolder shape, which required a holder to
// acquire WHILE RPCs were still in flight — a property neither Pebble's
// exclusive directory lock nor the fixed five-attempt backoff promises,
// and independent short-lived UI opens can legitimately reacquire
// between a holder's own attempts.
//
// The one direct Open call below either succeeds immediately or reveals
// a retained handle: a uiService caching a handle across calls would
// leave the lock held even after every RPC has returned, and this Open
// would then need retries or fail outright. No wall-clock assertion is
// made anywhere in this test — the property is proven by the STRUCTURAL
// absence of contention at the moment Open is attempted, not by timing
// how fast it returns.
func TestAHolderAcquiresTheStoreAfterConcurrentRPCsComplete(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
			errs[i] = err
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent RPC %d failed: %v", i, err)
		}
	}

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("holder Open after every concurrent RPC completed: %v — a retained handle would leave the lock held here", err)
	}
	defer holder.Close()
}

// captureDiagWriter redirects this package's diagnostic sink into a
// buffer for the duration of one test, restoring the previous writer on
// cleanup — the same capture shape internal/graphstore/logger_test.go
// uses for its own diagWriter seam.
func captureDiagWriter(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := setDiagWriter(&buf)
	t.Cleanup(func() { setDiagWriter(prev) })
	return &buf
}

// TestInternalErrorOnTheWireLeaksNoHostPath proves WR-01's mitigation at
// the same standard TestIndexingInProgressMessageLeaksNothing sets for
// the degrade path: an UNCLASSIFIED engine error — the arm of
// mapEngineError that used to place err.Error() verbatim on the wire —
// renders to the loopback browser caller as a fixed generic sentence
// carrying no path separator, while the operator still gets the real
// error on the server's own diagnostic stream.
//
// Both halves are asserted positively, so neither can pass vacuously:
//
//   - the wire message must EQUAL errInternal's text and contain no path
//     separator (containsPathSeparator is positive-controlled by
//     TestIndexingInProgressMessageLeaksNothing in this same file); and
//   - the captured diagnostic stream must contain the repository's own
//     absolute temp path, proving the detail was RELOCATED rather than
//     discarded. Removing the scrub turns the first half RED; removing
//     the writeDiagLine call turns the second half RED.
//
// The provoking scenario is the one handlers_test.go's
// "unreadable-candidate-within-the-cap" subtest already establishes:
// every multi-def candidate's source file is replaced by a directory of
// the same name, so the per-candidate read fails with an *os.PathError
// carrying the absolute host path.
func TestInternalErrorOnTheWireLeaksNoHostPath(t *testing.T) {
	dir := buildOverloadedFixture(t, "Leaky", 2)
	for i := 0; i < 2; i++ {
		p := filepath.Join(dir, fmt.Sprintf("pkg%d", i), fmt.Sprintf("pkg%d.go", i))
		if err := os.Remove(p); err != nil {
			t.Fatalf("remove %s: %v", p, err)
		}
		if err := os.Mkdir(p, 0o755); err != nil {
			t.Fatalf("mkdir %s (in place of the file): %v", p, err)
		}
	}

	diag := captureDiagWriter(t)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Leaky"}))
	if err == nil {
		t.Fatal("GetNodeDetail with every candidate's file replaced by a directory succeeded, want CodeInternal")
	}
	if code := connect.CodeOf(err); code != connect.CodeInternal {
		t.Fatalf("GetNodeDetail: code = %v, want CodeInternal", code)
	}

	connErr := &connect.Error{}
	if !errors.As(err, &connErr) {
		t.Fatalf("error %v is not a *connect.Error", err)
	}
	if got := connErr.Message(); got != errInternal.Error() {
		t.Fatalf("wire message = %q, want the fixed generic %q", got, errInternal.Error())
	}
	if containsPathSeparator(connErr.Message()) {
		t.Fatalf("wire message %q contains a path separator, want none — an unclassified os.PathError must never disclose the host checkout's layout", connErr.Message())
	}

	// The detail must have been relocated, not dropped: the operator's
	// diagnostic stream carries the real error, absolute path and all.
	logged := diag.String()
	if !strings.Contains(logged, dir) {
		t.Fatalf("diagnostic stream = %q, want it to contain the real error including the repository path %q — the scrub must relocate the detail server-side, never discard it", logged, dir)
	}
	if !strings.HasPrefix(logged, diagPrefix) {
		t.Fatalf("diagnostic stream = %q, want it to start with the provenance prefix %q", logged, diagPrefix)
	}
}
