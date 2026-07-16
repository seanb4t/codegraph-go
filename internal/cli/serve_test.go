package cli

import (
	"bytes"
	"strings"
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
// Reproduces the WATCH-02 regression directly, without racing the Go
// scheduler (WR-02, 03-REVIEW.md: the previous version asserted an ordering
// via an atomic flag the spawned goroutine could legitimately observe
// unset on GOMAXPROCS>1 — a false-positive flake against fully-correct
// production code). Instead, the hook's real work BLOCKS on a channel this
// test only closes AFTER serveWatchStart has returned:
//
//   - Correct code: serveWatchStart spawns the goroutine and returns while
//     the hook (the goroutine's first action) is parked on <-released; the
//     caller goroutine below then delivers the return value, we close
//     released, and the hook proceeds to signal workStarted. Deterministic —
//     no scheduling order can fail it.
//   - Mutated code (daemon.New / policy check / acquire / watch.Open moved
//     back above the goroutine boundary, i.e. the hook invoked synchronously
//     before the return): serveWatchStart deadlocks inside the hook waiting
//     for a close that only happens after it returns — the bounded select on
//     retCh below trips and fails the test. Still mutation-proof.
func TestServeWatchStartDeferred(t *testing.T) {
	_, main := statusWorktreeMismatchFixture(t)

	released := make(chan struct{})
	workStarted := make(chan struct{}, 1)
	onWatchWorkStart := func() {
		<-released
		select {
		case workStarted <- struct{}{}:
		default:
		}
	}

	type startResult struct {
		cancel func()
		done   <-chan struct{}
	}
	var stderr bytes.Buffer
	retCh := make(chan startResult, 1)
	go func() {
		cancel, done := serveWatchStart(main, true, false, false, indexer.Options{Quiet: true}, &stderr, onWatchWorkStart)
		retCh <- startResult{cancel: cancel, done: done}
	}()

	var res startResult
	select {
	case res = <-retCh:
	case <-time.After(5 * time.Second):
		t.Fatal("serveWatchStart did not return within 5s — it is blocked inside onWatchWorkStart, meaning the goroutine's real work (daemon.New/policy-check/acquire/watch.Open) ran synchronously BEFORE the return: the exact WATCH-02 regression this test exists to catch")
	}

	// serveWatchStart has returned; only now is the hook released to do the
	// goroutine's real work.
	close(released)

	select {
	case <-workStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("onWatchWorkStart never fired within 5s of release — serveWatchStart's goroutine did not start real work")
	}

	res.cancel()
	select {
	case <-res.done:
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

	// IN-03 (round 5): pin the FULL verbatim D-12 line — "[CodeGraph MCP] "
	// banner included — not just a reason substring: byte-identity with the
	// TS string is the stated contract, so a regression dropping the banner
	// or mangling the trailing guidance's punctuation must fail HERE, not
	// slip past a partial match.
	want := "[CodeGraph MCP] File watcher disabled — CODEGRAPH_NO_WATCH=1 is set. " +
		"The graph will not auto-update; run `codegraph sync` " +
		"(or install the git sync hooks via `codegraph init`) to refresh.\n"
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("expected the full verbatim disabled message on stderr,\nwant substring: %q\ngot: %q", want, stderr.String())
	}
}
