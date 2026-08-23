package uiserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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
