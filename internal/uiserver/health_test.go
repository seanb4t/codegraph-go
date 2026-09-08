package uiserver

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// uiserverWorktreeMismatchFixture REPLICATES the technique of
// internal/query/engine_worktree_test.go:48-70's worktreeMismatchFixture
// — it does not and cannot reuse it: worktreeMismatchFixture is
// unexported test code in package query, and a _test.go file in another
// package cannot import it at all. This is a deliberate, traceable copy
// of that fixture's git-worktree-creation steps, adapted to this
// package's own copyGofixture/runGitFixtureCmd/indexGofixture helpers
// rather than query's copyFixture/runGitW/indexFixture. If the two ever
// need to be shared, the correct move is a small internal test-support
// package — not exporting a test helper for one caller's convenience —
// and that is out of scope here.
func uiserverWorktreeMismatchFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()

	main := copyGofixture(t)
	runGitFixtureCmd(t, main, "init", "-q", ".")
	runGitFixtureCmd(t, main, "config", "user.email", "uiservertest@example.com")
	runGitFixtureCmd(t, main, "config", "user.name", "uiserver test")
	runGitFixtureCmd(t, main, "add", "-A")
	runGitFixtureCmd(t, main, "commit", "-q", "-m", "init")

	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitFixtureCmd(t, main, "worktree", "add", "-b", "probe", wt)

	indexGofixture(t, main)

	absMain, err := filepath.Abs(main)
	if err != nil {
		t.Fatalf("filepath.Abs(main): %v", err)
	}
	absWt, err := filepath.Abs(wt)
	if err != nil {
		t.Fatalf("filepath.Abs(wt): %v", err)
	}
	return absWt, absMain
}

// TestGetHealthProjectsStatusResult proves GetHealth carries every value
// HLT-01/HLT-02/HLT-03 need, compared field-for-field against an
// independently-computed eng.Status(ctx) call for the SAME fixture — not
// merely that the response is non-empty (RPC-01/RPC-02). Maps are
// compared entry-for-entry, not by length: a size-only assertion would
// pass on the wrong keys.
func TestGetHealthProjectsStatusResult(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.Status(context.Background())
	if err != nil {
		t.Fatalf("eng.Status (independent verification path): %v", err)
	}

	got := resp.Msg

	if got.GetInitialized() != want.Initialized {
		t.Errorf("initialized = %v, want %v", got.GetInitialized(), want.Initialized)
	}
	if got.GetVersion() != want.Version {
		t.Errorf("version = %q, want %q", got.GetVersion(), want.Version)
	}
	if got.GetFileCount() != want.FileCount {
		t.Errorf("file_count = %d, want %d", got.GetFileCount(), want.FileCount)
	}
	if got.GetNodeCount() != want.NodeCount {
		t.Errorf("node_count = %d, want %d", got.GetNodeCount(), want.NodeCount)
	}
	if got.GetEdgeCount() != want.EdgeCount {
		t.Errorf("edge_count = %d, want %d", got.GetEdgeCount(), want.EdgeCount)
	}
	// db_size_bytes is a live filesystem byte-sum over Pebble's
	// SSTables/WAL/MANIFEST (D-07) — it can differ by a few bytes between
	// two separate scans of the SAME live store (this RPC's internal
	// scan and this test's own independent verification Engine), so it
	// is checked for a sane positive value rather than byte-exact
	// equality, mirroring internal/query/files_status_test.go:518's own
	// ">  0, never erroring" convention for the identical field.
	if got.GetDbSizeBytes() <= 0 {
		t.Errorf("db_size_bytes = %d, want > 0 for a real indexed Pebble store", got.GetDbSizeBytes())
	}
	if got.GetBackend() != want.Backend {
		t.Errorf("backend = %q, want %q", got.GetBackend(), want.Backend)
	}
	if got.GetStale() != want.Stale {
		t.Errorf("stale = %v, want %v", got.GetStale(), want.Stale)
	}

	if len(want.FilesByLanguage) == 0 {
		t.Fatal("test fixture assumption broken: eng.Status's FilesByLanguage is empty")
	}
	if len(got.GetFilesByLanguage()) != len(want.FilesByLanguage) {
		t.Fatalf("files_by_language has %d entries, want %d", len(got.GetFilesByLanguage()), len(want.FilesByLanguage))
	}
	for lang, count := range want.FilesByLanguage {
		if got.GetFilesByLanguage()[lang] != count {
			t.Errorf("files_by_language[%q] = %d, want %d", lang, got.GetFilesByLanguage()[lang], count)
		}
	}

	if len(want.NodesByKind) == 0 {
		t.Fatal("test fixture assumption broken: eng.Status's NodesByKind is empty")
	}
	if len(got.GetNodesByKind()) != len(want.NodesByKind) {
		t.Fatalf("nodes_by_kind has %d entries, want %d", len(got.GetNodesByKind()), len(want.NodesByKind))
	}
	for kind, count := range want.NodesByKind {
		if got.GetNodesByKind()[kind] != count {
			t.Errorf("nodes_by_kind[%q] = %d, want %d", kind, got.GetNodesByKind()[kind], count)
		}
	}

	if len(want.EdgesByKind) == 0 {
		t.Fatal("test fixture assumption broken: eng.Status's EdgesByKind is empty")
	}
	if len(got.GetEdgesByKind()) != len(want.EdgesByKind) {
		t.Fatalf("edges_by_kind has %d entries, want %d", len(got.GetEdgesByKind()), len(want.EdgesByKind))
	}
	for kind, count := range want.EdgesByKind {
		if got.GetEdgesByKind()[kind] != count {
			t.Errorf("edges_by_kind[%q] = %d, want %d", kind, got.GetEdgesByKind()[kind], count)
		}
	}

	if len(want.Languages) == 0 {
		t.Fatal("test fixture assumption broken: eng.Status's Languages is empty")
	}
	if len(got.GetLanguages()) != len(want.Languages) {
		t.Fatalf("languages has %d entries, want %d", len(got.GetLanguages()), len(want.Languages))
	}
	for i, lang := range want.Languages {
		if got.GetLanguages()[i] != lang {
			t.Errorf("languages[%d] = %q, want %q", i, got.GetLanguages()[i], lang)
		}
	}

	// pending_changes is a documented inert placeholder — status.go's
	// StatusResult composite literal never assigns PendingChanges, and
	// files_status_test.go's "PendingChanges stays an inert placeholder"
	// subtest locks that as an engine-level invariant. Comparing against
	// want.PendingChanges (itself always the zero value) would be
	// tautological: it can never distinguish "correctly wired real
	// data" from "correctly wired always-zero placeholder", which is
	// exactly the gap CR-02 (04-REVIEW.md) found — this RPC put the
	// placeholder on the wire and the web page rendered it as if it
	// were live. Asserting the literal known-zero shape here, rather
	// than mirroring whatever want happens to hold, makes this test
	// fail the moment either side stops being zero — the wire mapping
	// silently starts fabricating a non-zero value, or the engine
	// starts computing a real one without this test being updated to
	// match — either of which is a case this test must not pass
	// silently through.
	pc := got.GetPendingChanges()
	if pc == nil {
		t.Fatal("pending_changes = nil, want a populated message")
	}
	if want.PendingChanges.Added != 0 || want.PendingChanges.Modified != 0 || want.PendingChanges.Removed != 0 {
		t.Fatalf("test fixture assumption broken: eng.Status's PendingChanges is no longer the documented all-zero placeholder (%+v) — update this test's expectations deliberately, not silently", want.PendingChanges)
	}
	if pc.GetAdded() != 0 || pc.GetModified() != 0 || pc.GetRemoved() != 0 {
		t.Errorf("pending_changes = %+v, want the documented all-zero placeholder {0,0,0}", pc)
	}

	ih := got.GetIndexHealth()
	if ih == nil {
		t.Fatal("index_health = nil, want a populated message")
	}
	if ih.GetBuiltWithVersion() != want.Index.BuiltWithVersion ||
		ih.GetBuiltWithExtractionVersion() != want.Index.BuiltWithExtractionVersion ||
		ih.GetCurrentExtractionVersion() != want.Index.CurrentExtractionVersion ||
		ih.GetReindexRecommended() != want.Index.ReindexRecommended ||
		ih.GetState() != want.Index.State ||
		int(ih.GetPendingRefs()) != want.Index.PendingRefs {
		t.Errorf("index_health = %+v, want %+v", ih, want.Index)
	}
}

// TestGetHealthWorktreeMismatchPopulated is the positive direction: an
// Engine whose start path is a linked git worktree, querying the main
// checkout's index, returns a non-nil worktree_mismatch naming both
// roots.
func TestGetHealthWorktreeMismatchPopulated(t *testing.T) {
	wt, main := uiserverWorktreeMismatchFixture(t)

	srv := startedServer(t, wt)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}

	mismatch := resp.Msg.GetWorktreeMismatch()
	if mismatch == nil {
		t.Fatal("worktree_mismatch = nil, want a populated mismatch (worktree queries the main checkout's index)")
	}

	wantIndexRoot, err := filepath.EvalSymlinks(main)
	if err != nil {
		t.Fatalf("EvalSymlinks(main): %v", err)
	}
	wantWorktreeRoot, err := filepath.EvalSymlinks(wt)
	if err != nil {
		t.Fatalf("EvalSymlinks(wt): %v", err)
	}
	if mismatch.GetIndexRoot() != wantIndexRoot {
		t.Errorf("worktree_mismatch.index_root = %q, want %q", mismatch.GetIndexRoot(), wantIndexRoot)
	}
	if mismatch.GetWorktreeRoot() != wantWorktreeRoot {
		t.Errorf("worktree_mismatch.worktree_root = %q, want %q", mismatch.GetWorktreeRoot(), wantWorktreeRoot)
	}
}

// TestGetHealthWorktreeMismatchNilInTree is the REQUIRED negative
// direction (a populated-only test would pass while the nil case
// silently regresses into a false warning on every clean repository): an
// ordinary, in-tree Engine returns a nil worktree_mismatch.
func TestGetHealthWorktreeMismatchNilInTree(t *testing.T) {
	dir, _ := newGitIndexedFixture(t)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}

	if mismatch := resp.Msg.GetWorktreeMismatch(); mismatch != nil {
		t.Fatalf("worktree_mismatch = %+v, want nil for an in-tree Engine", mismatch)
	}
}

// TestGetHealthCommitShaValidated proves T-04-11: a Meta record carrying
// a malformed commit SHA (simulated the same way
// TestUIServiceStatusCarriesCommitSHA's malformed-SHA subtest does, via
// overwriteCommitSHA which bypasses write-time validation) degrades to
// an empty commit_sha rather than leaving the malformed value on the
// wire.
func TestGetHealthCommitShaValidated(t *testing.T) {
	dir, _ := newGitIndexedFixture(t)
	overwriteCommitSHA(t, dir, "../../attacker/attacker-repo/blob/main")

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if got := resp.Msg.GetCommitSha(); got != "" {
		t.Fatalf("commit_sha = %q, want empty — a malformed stored commit_sha must never leave this server", got)
	}
}

// TestGetHealthOpensEngineExactlyOnce proves the withEngine open-use-close
// discipline (SRV-04): one GetHealth call invokes the package's openEngine
// seam exactly once, using the same counting-wrapper technique
// TestUIServiceOpensThroughTheSingleSeam already establishes in
// handlers_test.go.
func TestGetHealthOpensEngineExactlyOnce(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	orig := openEngine
	t.Cleanup(func() { openEngine = orig })
	var opens int64
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		atomic.AddInt64(&opens, 1)
		return orig(start)
	}

	if _, err := client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{})); err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if got := atomic.LoadInt64(&opens); got != 1 {
		t.Fatalf("openEngine invoked %d times for one GetHealth call, want exactly 1", got)
	}
}

// TestGetHealthDoesNotDegrade proves GetHealth is NOT GetStatus: with the
// store held past graphstore.Open's bounded retry budget by a REAL
// graphstore.Open handle in this test process (the same technique
// TestStatusDegradesOnLock uses to prove GetStatus's OPPOSITE behavior),
// GetHealth returns an ERROR through mapEngineError — never a
// successful degraded response.
func TestGetHealthDoesNotDegrade(t *testing.T) {
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

	_, err = client.GetHealth(context.Background(), connect.NewRequest(&uiv1.GetHealthRequest{}))
	if err == nil {
		t.Fatal("GetHealth while the store is locked past the retry budget succeeded, want an error (GetHealth must not degrade-and-answer)")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnavailable {
		t.Fatalf("GetHealth while locked: code = %v, want CodeUnavailable", code)
	}
}
