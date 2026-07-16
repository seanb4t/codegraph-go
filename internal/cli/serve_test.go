package cli

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// TestServeKeepsStartPathDistinctFromConfinementRoot is WR-01's required
// test (02-REVIEW-2.md): it exercises serveServerPaths, the EXACT function
// newServeCmd's RunE calls to derive BuildServer's repoPath argument — not
// a hand-built replica living only in a test file (markdown_test.go's
// deriveServeRepoPath, in internal/mcp, is such a replica; it proves the
// mismatch fixture is wired correctly, but nothing there observes whether
// serve.go itself still performs this derivation).
//
// Reproduces the reviewer's mutation directly: if serveServerPaths ever
// collapsed to `return start, hasIndex, nil` unconditionally (the literal
// CR-01 regression — BuildServer(..., repoPath, repoPath)), repoPath would
// equal wt here and this test would fail.
func TestServeKeepsStartPathDistinctFromConfinementRoot(t *testing.T) {
	wt, main := statusWorktreeMismatchFixture(t)

	repoPath, hasIndex, err := serveServerPaths(wt)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", wt, err)
	}
	if !hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = false, want true (main checkout was indexed)", wt)
	}
	if repoPath == wt {
		t.Fatal("serveServerPaths returned the START path as repoPath — CR-01 regression: repoPath must be the RESOLVED index root, distinct from the caller's start path in a worktree, or BuildServer's worktree-mismatch detection silently short-circuits to nil on every production call")
	}
	if repoPath != main {
		t.Fatalf("serveServerPaths(%s) repoPath = %q, want the main checkout %q", wt, repoPath, main)
	}
}

// TestServeServerPathsNoIndex pins MCP-03: when no .codegraph/ resolves
// above start, serveServerPaths reports hasIndex=false (not an error) and
// repoPath falls back to start itself — serve still starts with zero tools
// rather than refusing the connection.
func TestServeServerPathsNoIndex(t *testing.T) {
	dir := t.TempDir()

	repoPath, hasIndex, err := serveServerPaths(dir)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", dir, err)
	}
	if hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = true, want false (no .codegraph/ present)", dir)
	}
	if repoPath != dir {
		t.Fatalf("serveServerPaths(%s) repoPath = %q, want %q (no index found: repoPath falls back to start)", dir, repoPath, dir)
	}
}

// TestServeWatchStartDeferred is WATCH-02's mutation-proof structural test
// (D-08): it exercises the REAL serveWatchStart the RunE body calls, not a
// hand-built replica living only in this test file. It proves
// serveWatchStart returns to its caller BEFORE any of daemon.New / the
// watch-policy check (inside daemon.Run) / lock acquisition / watch.Open's
// recursive fsnotify walk executes inside the goroutine it spawns, via a
// deterministic synchronization hook (onWatchWorkStart) rather than a
// sleep/timeout race — mirroring daemon.Daemon's onSyncStart precedent
// (internal/daemon/daemon.go lines 67-75).
//
// Reproduces the WATCH-02 regression directly: if daemon.New (or the policy
// check, acquire, or watch.Open) were ever moved back above serveWatchStart's
// goroutine boundary — called synchronously before the goroutine spawn,
// rather than as the first thing the goroutine itself does — the
// onWatchWorkStart hook would observe `returned` still false (the goroutine's
// real work would have started before serveWatchStart's return), and the
// happens-before assertion below would fail this test.
func TestServeWatchStartDeferred(t *testing.T) {
	_, main := statusWorktreeMismatchFixture(t)

	var returned int32
	workStarted := make(chan struct{}, 1)
	onWatchWorkStart := func() {
		// If serveWatchStart has not yet returned by the time this fires,
		// the goroutine performed real work (daemon.New/policy/acquire/
		// watch.Open) BEFORE its synchronous return — the exact WATCH-02
		// regression this test exists to catch.
		if atomic.LoadInt32(&returned) == 0 {
			t.Error("serveWatchStart's goroutine started real work BEFORE serveWatchStart returned to its caller — WATCH-02 regression: daemon.New/policy-check/acquire/watch.Open must run strictly after the return, inside the spawned goroutine")
		}
		select {
		case workStarted <- struct{}{}:
		default:
		}
	}

	var stderr bytes.Buffer
	cancel, done := serveWatchStart(main, true, false, false, indexer.Options{Quiet: true}, &stderr, onWatchWorkStart)
	// The marker is set immediately after the call returns — establishing
	// the happens-before ordering onWatchWorkStart checks above.
	atomic.StoreInt32(&returned, 1)

	select {
	case <-workStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("onWatchWorkStart never fired within 5s — serveWatchStart's goroutine did not start real work")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serveWatchStart's goroutine did not join within 5s of cancel() — leaked goroutine")
	}
}

// TestServeWatchStartDisabledPrintsVerbatimMessage pins RESEARCH.md's
// Pitfall 2: --no-watch must NOT silently skip watcher startup — it still
// spawns the goroutine and routes noWatch through the SAME
// watch.WatchDisabledReason check CODEGRAPH_NO_WATCH=1 would, printing the
// verbatim D-12/D-13 TS-parity disabled message to the provided stderr
// writer rather than swallowing it via an early return.
func TestServeWatchStartDisabledPrintsVerbatimMessage(t *testing.T) {
	_, main := statusWorktreeMismatchFixture(t)

	var stderr bytes.Buffer
	cancel, done := serveWatchStart(main, true, true, false, indexer.Options{Quiet: true}, &stderr, nil)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serveWatchStart's goroutine did not join within 5s of cancel() — leaked goroutine")
	}

	if !strings.Contains(stderr.String(), "File watcher disabled — CODEGRAPH_NO_WATCH=1 is set") {
		t.Fatalf("expected verbatim disabled message on stderr, got: %q", stderr.String())
	}
}
