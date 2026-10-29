package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/wtindex"
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

	indexRoot, repoPath, hasIndex, err := serveServerPaths(wt)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", wt, err)
	}
	if !hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = false, want true (main checkout was indexed)", wt)
	}
	if repoPath == wt {
		t.Fatal("serveServerPaths returned the START path as repoPath — CR-01 regression: repoPath must be the RESOLVED index root, distinct from the caller's start path in a worktree, or BuildServer's worktree-mismatch detection silently short-circuits to nil on every production call")
	}
	if repoPath != main || indexRoot != main {
		t.Fatalf("serveServerPaths(%s) = index root %q, confinement root %q, want the main checkout %q for both", wt, indexRoot, repoPath, main)
	}
}

// TestServeServerPathsSiblingWorktree pins the sibling case: a linked
// worktree outside the main checkout syncs and watches the main checkout's
// index, while its confinement root is the worktree itself, so the default
// path (a subdirectory of it) stays inside.
func TestServeServerPathsSiblingWorktree(t *testing.T) {
	wt, main := siblingWorktreeFixture(t)
	realMain, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", main, err)
	}
	start := filepath.Join(wt, "pkga")

	indexRoot, confineRoot, hasIndex, err := serveServerPaths(start)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", start, err)
	}
	if !hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = false, want true (the main checkout is indexed)", start)
	}
	if indexRoot != realMain {
		t.Fatalf("serveServerPaths(%s) index root = %q, want the main checkout %q", start, indexRoot, realMain)
	}
	if confineRoot != wt {
		t.Fatalf("serveServerPaths(%s) confinement root = %q, want the worktree %q", start, confineRoot, wt)
	}
}

// TestServeServerPathsNoIndex pins MCP-03: when no .codegraph/ resolves
// above start, serveServerPaths reports hasIndex=false (not an error) and
// repoPath falls back to start itself — serve still starts with zero tools
// rather than refusing the connection.
func TestServeServerPathsNoIndex(t *testing.T) {
	dir := t.TempDir()

	indexRoot, repoPath, hasIndex, err := serveServerPaths(dir)
	if err != nil {
		t.Fatalf("serveServerPaths(%s): unexpected error: %v", dir, err)
	}
	if hasIndex {
		t.Fatalf("serveServerPaths(%s) hasIndex = true, want false (no .codegraph/ present)", dir)
	}
	if repoPath != dir || indexRoot != dir {
		t.Fatalf("serveServerPaths(%s) = index root %q, confinement root %q, want %q for both (no index found: both fall back to start)", dir, indexRoot, repoPath, dir)
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
//
// The "pending worktree build" case does the same with a build that
// serveWatchStart must finish inside the goroutine: the worktree index must
// not exist when serveWatchStart returns.
func TestServeWatchStartDeferred(t *testing.T) {
	t.Run("watcher", func(t *testing.T) {
		_, main := statusWorktreeMismatchFixture(t)
		testServeWatchStartDeferred(t, main, nil, "")
	})
	t.Run("pending worktree build", func(t *testing.T) {
		t.Setenv("CODEGRAPH_NO_WATCH", "")
		wt, indexRoot, pending := pendingWorktreeBuild(t)
		testServeWatchStartDeferred(t, indexRoot, pending, wt)
	})
}

func testServeWatchStartDeferred(t *testing.T, main string, pending *wtindex.Build, wt string) {
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
	stderr := newSyncWriter()
	retCh := make(chan startResult, 1)
	go func() {
		cancel, done := serveWatchStart(main, true, pending, false, pending != nil, indexer.Options{Quiet: true}, stderr, onWatchWorkStart)
		retCh <- startResult{cancel: cancel, done: done}
	}()

	var res startResult
	select {
	case res = <-retCh:
	case <-time.After(5 * time.Second):
		t.Fatal("serveWatchStart did not return within 5s — it is blocked inside onWatchWorkStart, meaning the goroutine's real work (daemon.New/policy-check/acquire/watch.Open) ran synchronously BEFORE the return: the exact WATCH-02 regression this test exists to catch")
	}
	if pending != nil && pathExists(t, filepath.Join(wt, codegraphDirName)) {
		t.Fatal("the worktree index exists when serveWatchStart returns: the build ran above the goroutine, on the handshake path")
	}

	// serveWatchStart has returned; only now is the hook released to do the
	// goroutine's real work.
	close(released)

	select {
	case <-workStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("onWatchWorkStart never fired within 5s of release — serveWatchStart's goroutine did not start real work")
	}
	if pending != nil {
		awaitDaemonLock(t, wt, stderr)
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
// documented D-12/D-13 disabled message to the provided stderr
// writer rather than swallowing it via an early return.
func TestServeWatchStartDisabledPrintsVerbatimMessage(t *testing.T) {
	_, main := statusWorktreeMismatchFixture(t)

	var stderr bytes.Buffer
	cancel, done := serveWatchStart(main, true, nil, true, false, indexer.Options{Quiet: true}, &stderr, nil)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("serveWatchStart's goroutine did not join within 5s of cancel() — leaked goroutine")
	}

	// IN-03 (round 5): pin the FULL verbatim D-12 line — "[CodeGraph MCP] "
	// banner included — not just a reason substring: byte-identity with the
	// documented message is the stated contract, so a regression dropping
	// the banner or mangling the trailing guidance's punctuation must fail
	// HERE, not slip past a partial match.
	want := "[CodeGraph MCP] File watcher disabled — CODEGRAPH_NO_WATCH=1 is set. " +
		"The graph will not auto-update; run `codegraph sync` " +
		"(or install the git sync hooks via `codegraph init`) to refresh.\n"
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("expected the full verbatim disabled message on stderr,\nwant substring: %q\ngot: %q", want, stderr.String())
	}
}

// TestServeServerPathsIgnoresHomeCodegraph: a .codegraph in the home directory (older releases kept the
// daemon registry there, and other tools may still create one) must not
// make serveServerPaths report an index for the home directory or for an
// unindexed repository under it. The store check at the end is the
// acceptance criterion in words: nothing under the fake home was opened
// or built as a project.
func TestServeServerPathsIgnoresHomeCodegraph(t *testing.T) {
	home := fakeHome(t)
	if err := os.MkdirAll(filepath.Join(home, codegraphDirName, "daemons"), 0o755); err != nil {
		t.Fatalf("mkdir fake ~/.codegraph/daemons: %v", err)
	}
	repo := filepath.Join(home, "x", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	for _, start := range []string{home, repo} {
		indexRoot, repoPath, hasIndex, err := serveServerPaths(start)
		if err != nil {
			t.Fatalf("serveServerPaths(%s): unexpected error: %v", start, err)
		}
		if hasIndex {
			t.Fatalf("serveServerPaths(%s) hasIndex = true, want false (~/.codegraph must not count as an index)", start)
		}
		if repoPath != start || indexRoot != start {
			t.Fatalf("serveServerPaths(%s) = index root %q, confinement root %q, want %q for both (no index found: both fall back to start)", start, indexRoot, repoPath, start)
		}
	}
	if _, err := os.Stat(filepath.Join(home, codegraphDirName, storeDirName)); !os.IsNotExist(err) {
		t.Fatalf("~/.codegraph/store exists after serveServerPaths (stat err %v); the home directory was treated as a project", err)
	}
}
