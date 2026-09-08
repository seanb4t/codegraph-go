package uiserver

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// runGitFixtureCmd runs a git subcommand against dir, failing the test on
// any error. Test-only fixture-building helper — never the code path
// under test, which is the RPC layer's commit_sha mapping.
func runGitFixtureCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v (in %s): %v\n%s", args, dir, err, out)
	}
}

// gitRevParseHEAD runs `git rev-parse HEAD` against dir directly — NEVER
// through the code path under test — so a comparison against it is a
// comparison to an independent oracle, mirroring
// internal/indexer/commit_test.go's identically-named helper.
func gitRevParseHEAD(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("independent git rev-parse HEAD in %s: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

// newGitIndexedFixture git-inits a fresh temp directory, commits one
// minimal Go module, indexes it, and returns (repoDir, wantCommitSHA) —
// wantCommitSHA computed via an independent `git rev-parse HEAD`
// subprocess (gitRevParseHEAD), never by reading back whatever the
// indexer itself wrote to Meta.
func newGitIndexedFixture(t *testing.T) (dir string, wantSHA string) {
	t.Helper()
	dir = t.TempDir()
	runGitFixtureCmd(t, dir, "init", "-q", ".")
	runGitFixtureCmd(t, dir, "config", "user.email", "uiservertest@example.com")
	runGitFixtureCmd(t, dir, "config", "user.name", "uiserver test")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/uiservertest\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	runGitFixtureCmd(t, dir, "add", "-A")
	runGitFixtureCmd(t, dir, "commit", "-q", "-m", "init")

	wantSHA = gitRevParseHEAD(t, dir)

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	return dir, wantSHA
}

// startedServer builds a live server over an indexed repoPath and starts
// Serve in a background goroutine, returning once the port is
// connectable. Shared scaffolding for every wire-driven test below,
// mirroring server_test.go's own mustListen/waitForConnectable sequence.
func startedServer(t *testing.T, repoPath string) *Server {
	t.Helper()
	srv := mustListen(t, repoPath)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Close()
	})
	go func() { _ = srv.Serve(ctx) }()
	waitForConnectable(t, strings.TrimPrefix(srv.URL(), "http://"))
	return srv
}

// TestUIServiceSearchMatchesEngine drives Search through a real
// uiv1connect client against a real listener and compares the result
// element-by-element against an independently-computed Engine.Search
// call for the same arguments — not merely that the response is
// non-empty (RPC-01/RPC-02).
func TestUIServiceSearchMatchesEngine(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha"}))
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	// Depth/limit arguments pass straight through to the Engine, which
	// already bounds them (validateLimit/MaxLimit) for every caller — the
	// RPC adds no second copy of that rule. For query.Search specifically,
	// validateLimit REJECTS an explicit limit argument above MaxLimit
	// outright (it does not silently clamp an out-of-range REQUEST — only
	// a naturally-oversized RESULT for an in-range request is clamped, by
	// the len(locations) > MaxLimit guard further down the same method).
	// The claim this test proves is narrower and causal either way: the
	// RPC must classify an over-MaxLimit limit EXACTLY as Engine.Search
	// itself does for the identical arguments, never by a second,
	// independently-drifting check at this layer. Run BEFORE opening the
	// independent verification Engine below — two concurrent
	// query.OpenAt calls against the SAME store directory (this RPC's own
	// internal open, and this test's own verification open) are
	// lock-contended, and asserting a Connect code against a store-lock
	// failure would prove nothing about limit classification.
	overLimit := int32(query.MaxLimit) + 1
	if _, err := client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha", Limit: overLimit})); err == nil {
		t.Fatal("Search with limit above MaxLimit succeeded over the wire, want the Engine's own rejection reflected unchanged")
	} else if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("Search with limit above MaxLimit: code = %v, want CodeInvalidArgument", code)
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.Search("Alpha", "", 0)
	if err != nil {
		t.Fatalf("eng.Search (independent verification path): %v", err)
	}
	if len(want) == 0 {
		t.Fatal("test fixture assumption broken: eng.Search(\"Alpha\", ...) returned no matches")
	}

	got := resp.Msg.GetLocations()
	if len(got) != len(want) {
		t.Fatalf("Search returned %d locations, want %d (independently computed by eng.Search)", len(got), len(want))
	}
	for i := range want {
		if got[i].GetName() != want[i].Name ||
			got[i].GetKind() != want[i].Kind ||
			got[i].GetFilePath() != want[i].FilePath ||
			got[i].GetStartLine() != want[i].StartLine {
			t.Fatalf("Search location[%d] = %+v, want %+v (independently computed by eng.Search)", i, got[i], want[i])
		}
	}

	if _, err := eng.Search("Alpha", "", int(overLimit)); err == nil {
		t.Fatal("test assumption broken: independently-computed eng.Search with the same over-limit argument succeeded, want an error")
	}
}

// TestUIServiceStatusCarriesCommitSHA proves ENG-04/D-05/D-06's data
// path: a git-checkout fixture's GetStatus response carries a non-empty
// commit_sha equal to that fixture's own `git rev-parse HEAD` (computed
// independently, never read back from what the indexer itself wrote),
// and a non-git fixture yields an empty commit_sha alongside a
// successful, initialized response.
func TestUIServiceStatusCarriesCommitSHA(t *testing.T) {
	t.Run("git-checkout", func(t *testing.T) {
		dir, wantSHA := newGitIndexedFixture(t)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
		if err != nil {
			t.Fatalf("GetStatus: %v", err)
		}
		if resp.Msg.GetCommitSha() == "" {
			t.Fatal("GetStatus commit_sha is empty for a git checkout, want the fixture's own HEAD")
		}
		if resp.Msg.GetCommitSha() != wantSHA {
			t.Fatalf("GetStatus commit_sha = %q, want %q (independent git rev-parse HEAD)", resp.Msg.GetCommitSha(), wantSHA)
		}
	})

	t.Run("non-git-tree", func(t *testing.T) {
		dir := copyGofixture(t) // t.TempDir(), never git-inited
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
		if err != nil {
			t.Fatalf("GetStatus: %v", err)
		}
		if !resp.Msg.GetInitialized() {
			t.Fatal("GetStatus initialized = false, want true for a successful response")
		}
		if resp.Msg.GetCommitSha() != "" {
			t.Fatalf("GetStatus commit_sha = %q, want empty for a non-git fixture", resp.Msg.GetCommitSha())
		}
	})

	// IN-06: GetStatus reads Meta.commit_sha through
	// schema.IndexedCommitSHA exactly like GetPermalink does, but a Meta
	// record on disk is not bound by internal/indexer's write-time
	// validation — a store built or edited by anything other than this
	// binary's own indexer is not covered by that guarantee.
	// overwriteCommitSHA (permalink_test.go) bypasses write-time
	// validation directly to simulate that. Reuses the malformed value
	// GetPermalink's own malformed-SHA test drives, so the two guards
	// stay aligned on what "malformed" means.
	t.Run("malformed-commit-sha-degrades-to-empty", func(t *testing.T) {
		dir, _ := newGitIndexedFixture(t)
		overwriteCommitSHA(t, dir, "../../attacker/attacker-repo/blob/main")
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{}))
		if err != nil {
			t.Fatalf("GetStatus: %v", err)
		}
		if !resp.Msg.GetInitialized() {
			t.Fatal("GetStatus initialized = false, want true for a successful response")
		}
		if got := resp.Msg.GetCommitSha(); got != "" {
			t.Fatalf("GetStatus commit_sha = %q, want empty — a malformed stored commit_sha must never leave this server", got)
		}
	})
}

// TestUIServiceErrorClassesAreTyped proves mapEngineError's classification
// is typed, not string-matched: query.ErrNotFound classifies as
// connect.CodeNotFound and query.ErrInvalidArgument as
// connect.CodeInvalidArgument, both asserted via connect.CodeOf(err).
// The invalid-argument case is additionally driven through a real
// uiv1connect client (Search's own ValidateKind rejection) so at least
// one classification is proven over the real wire, not only in-process;
// the not-found case is asserted directly against mapEngineError itself,
// since no rpc in this plan's Task 1 surface (GetStatus, Search) has a
// symbol-resolution path that can produce one — Callers/Callees/Impact/
// Affected, which do, are Task 3's addition. Both are equally valid
// homes for this task's own artifact under test (mapEngineError): a
// no-op replacement of either branch turns its subtest RED on the code
// it asserts.
func TestUIServiceErrorClassesAreTyped(t *testing.T) {
	t.Run("not-found", func(t *testing.T) {
		err := mapEngineError(query.ErrNotFound)
		if code := connect.CodeOf(err); code != connect.CodeNotFound {
			t.Fatalf("mapEngineError(query.ErrNotFound) code = %v, want CodeNotFound", code)
		}
	})

	t.Run("invalid-argument", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		_, err := client.Search(context.Background(), connect.NewRequest(&uiv1.SearchRequest{Term: "Alpha", Kind: "not-a-real-kind"}))
		if err == nil {
			t.Fatal("Search with an unknown kind succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("Search with an unknown kind: code = %v, want CodeInvalidArgument", code)
		}
	})
}

// TestUIServiceOpensThroughTheSingleSeam swaps the package-level
// openEngine var for a counting wrapper and asserts exactly one open per
// RPC call — the structural proof, alongside the positive-controlled
// query.OpenAt/withEngine count gates, that SRV-04's per-call
// open-snapshot-close discipline holds for the RPCs this task adds.
func TestUIServiceOpensThroughTheSingleSeam(t *testing.T) {
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

	if _, err := client.GetStatus(context.Background(), connect.NewRequest(&uiv1.GetStatusRequest{})); err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if got := atomic.LoadInt64(&opens); got != 1 {
		t.Fatalf("openEngine invoked %d times for one GetStatus call, want exactly 1", got)
	}
}

// findFileTreeChild returns the child of nodes named name, or nil.
func findFileTreeChild(nodes []*uiv1.FileTreeNode, name string) *uiv1.FileTreeNode {
	for _, n := range nodes {
		if n.GetName() == name {
			return n
		}
	}
	return nil
}

// TestUIServiceFilesPreservesBothFormats proves FilesResult's union
// contract survives the wire: a flat request returns a populated entry
// list and an EMPTY tree; a tree request returns a populated tree and an
// EMPTY entry list; both compared against an independently-computed
// Engine.Files call for the same options. The third subtest walks the
// tree at least two levels deep and asserts a directory node carries
// children and no path while a leaf carries a path and language and no
// children — FileTreeNode's documented asymmetry.
func TestUIServiceFilesPreservesBothFormats(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	// openIndependentEngine opens/computes/closes its own Engine per call
	// rather than sharing one held open across subtests: an Engine held
	// open for the whole test would lock-contend with each subtest's own
	// RPC call, which internally opens/closes the SAME store directory
	// (SRV-04's per-call discipline) — a store-lock failure there would
	// prove nothing about the union contract this test actually checks.
	openIndependentEngine := func(t *testing.T, opts query.FilesOptions) query.FilesResult {
		t.Helper()
		eng, closer, err := query.OpenAt(dir)
		if err != nil {
			t.Fatalf("query.OpenAt (independent verification path): %v", err)
		}
		defer closer.Close()
		result, err := eng.Files(opts)
		if err != nil {
			t.Fatalf("eng.Files (independent verification path): %v", err)
		}
		return result
	}

	t.Run("flat", func(t *testing.T) {
		resp, err := client.Files(context.Background(), connect.NewRequest(&uiv1.FilesRequest{Format: "flat"}))
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		want := openIndependentEngine(t, query.FilesOptions{Format: "flat"})
		if len(want.Files) == 0 {
			t.Fatal("test fixture assumption broken: eng.Files(flat) returned no entries")
		}
		if len(resp.Msg.GetFiles()) != len(want.Files) {
			t.Fatalf("Files(flat) returned %d entries, want %d (independently computed)", len(resp.Msg.GetFiles()), len(want.Files))
		}
		if len(resp.Msg.GetTree()) != 0 {
			t.Fatalf("Files(flat) tree has %d entries, want 0 (empty for the flat format)", len(resp.Msg.GetTree()))
		}
		for i, wantEntry := range want.Files {
			got := resp.Msg.GetFiles()[i]
			if got.GetPath() != wantEntry.Path || got.GetLanguage() != wantEntry.Language ||
				got.GetNodeCount() != wantEntry.NodeCount || got.GetEdgeCount() != wantEntry.EdgeCount {
				t.Fatalf("Files(flat) entry[%d] = %+v, want %+v", i, got, wantEntry)
			}
		}
	})

	t.Run("tree", func(t *testing.T) {
		resp, err := client.Files(context.Background(), connect.NewRequest(&uiv1.FilesRequest{Format: "tree"}))
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		want := openIndependentEngine(t, query.FilesOptions{Format: "tree"})
		if len(want.Tree) == 0 {
			t.Fatal("test fixture assumption broken: eng.Files(tree) returned no tree nodes")
		}
		if len(resp.Msg.GetFiles()) != 0 {
			t.Fatalf("Files(tree) files has %d entries, want 0 (empty for the tree format)", len(resp.Msg.GetFiles()))
		}
		if len(resp.Msg.GetTree()) != len(want.Tree) {
			t.Fatalf("Files(tree) returned %d root nodes, want %d (independently computed)", len(resp.Msg.GetTree()), len(want.Tree))
		}
	})

	t.Run("tree-children-round-trip", func(t *testing.T) {
		resp, err := client.Files(context.Background(), connect.NewRequest(&uiv1.FilesRequest{Format: "tree"}))
		if err != nil {
			t.Fatalf("Files: %v", err)
		}
		// The gofixture corpus's pkga/pkga.go and pkga/embed.go put a
		// "pkga" directory at least two levels deep (root -> pkga/ ->
		// pkga.go): exactly the shape this subtest needs to walk.
		pkgaDir := findFileTreeChild(resp.Msg.GetTree(), "pkga")
		if pkgaDir == nil {
			t.Fatal("test fixture assumption broken: no \"pkga\" directory at tree root")
		}
		if !pkgaDir.GetIsDir() {
			t.Fatalf("pkga node IsDir = false, want true")
		}
		if pkgaDir.GetPath() != "" {
			t.Fatalf("pkga directory node Path = %q, want empty (directories carry no path)", pkgaDir.GetPath())
		}
		if len(pkgaDir.GetChildren()) == 0 {
			t.Fatal("pkga directory node has no children, want at least one file")
		}

		leaf := findFileTreeChild(pkgaDir.GetChildren(), "pkga.go")
		if leaf == nil {
			t.Fatal("test fixture assumption broken: no \"pkga.go\" leaf under the pkga directory")
		}
		if leaf.GetIsDir() {
			t.Fatalf("pkga.go node IsDir = true, want false (a file leaf)")
		}
		if leaf.GetPath() == "" {
			t.Fatal("pkga.go leaf node Path is empty, want a non-empty file path")
		}
		if leaf.GetLanguage() == "" {
			t.Fatal("pkga.go leaf node Language is empty, want a non-empty language")
		}
		if len(leaf.GetChildren()) != 0 {
			t.Fatalf("pkga.go leaf node has %d children, want 0 (leaves carry no children)", len(leaf.GetChildren()))
		}
	})
}

// TestUIServiceFilesRejectsInvalidInput proves Files reflects the
// Engine's own input rejections as CodeInvalidArgument rather than an
// empty successful response — the RPC layer adds no second copy of
// either rule and does not silently swallow a rejection.
func TestUIServiceFilesRejectsInvalidInput(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	t.Run("invalid-format", func(t *testing.T) {
		_, err := client.Files(context.Background(), connect.NewRequest(&uiv1.FilesRequest{Format: "not-a-real-format"}))
		if err == nil {
			t.Fatal("Files with an unknown format succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("Files with an unknown format: code = %v, want CodeInvalidArgument", code)
		}
	})

	t.Run("depth-above-maximum", func(t *testing.T) {
		_, err := client.Files(context.Background(), connect.NewRequest(&uiv1.FilesRequest{Depth: int32(query.MaxDepth) + 1}))
		if err == nil {
			t.Fatal("Files with depth above the Engine's maximum succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("Files with depth above the Engine's maximum: code = %v, want CodeInvalidArgument", code)
		}
	})
}

// TestUIServiceTraversalsMatchEngine proves the four remaining
// structured-result reads — Callers, Callees, Impact, Affected — answer
// over the wire exactly as the corresponding Engine method does for the
// same arguments, computed independently in this test rather than merely
// asserted non-empty. It also proves depth/limit bounds are the Engine's
// alone (never a second, independently-drifting copy at this layer),
// that Affected accepts more than one file path, and that an unknown
// symbol classifies as CodeNotFound rather than CodeInternal.
//
// The gofixture corpus's pkga.Alpha calls the unexported pkga.helper
// (internal/indexer/testdata/gofixture/pkga/pkga.go), giving a real,
// deterministic caller/callee pair to drive Callers/Callees/Impact
// against.
func TestUIServiceTraversalsMatchEngine(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	// openIndependentEngine opens a fresh Engine for exactly one caller's
	// use, to be closed before the NEXT rpc call in the same subtest —
	// never held open across an RPC call, which would lock-contend with
	// that call's own internal open (SRV-04's per-call discipline; see
	// TestUIServiceFilesPreservesBothFormats's openIndependentEngine for
	// the same reasoning).
	openIndependentEngine := func(t *testing.T) (*query.Engine, func()) {
		t.Helper()
		eng, closer, err := query.OpenAt(dir)
		if err != nil {
			t.Fatalf("query.OpenAt (independent verification path): %v", err)
		}
		return eng, func() { closer.Close() }
	}

	t.Run("callers", func(t *testing.T) {
		resp, err := client.Callers(context.Background(), connect.NewRequest(&uiv1.CallersRequest{Symbol: "helper"}))
		if err != nil {
			t.Fatalf("Callers: %v", err)
		}
		eng, closeFn := openIndependentEngine(t)
		defer closeFn()
		want, err := eng.Callers("helper", 0)
		if err != nil {
			t.Fatalf("eng.Callers (independent verification path): %v", err)
		}
		if len(want.Callers) == 0 {
			t.Fatal("test fixture assumption broken: eng.Callers(\"helper\") returned no callers")
		}
		if resp.Msg.GetSymbol() != want.Symbol {
			t.Fatalf("Callers symbol = %q, want %q", resp.Msg.GetSymbol(), want.Symbol)
		}
		got := resp.Msg.GetCallers()
		if len(got) != len(want.Callers) {
			t.Fatalf("Callers returned %d locations, want %d (independently computed)", len(got), len(want.Callers))
		}
		for i, w := range want.Callers {
			g := got[i]
			if g.GetName() != w.Name || g.GetKind() != w.Kind || g.GetFilePath() != w.FilePath || g.GetStartLine() != w.StartLine {
				t.Fatalf("Callers location[%d] = %+v, want %+v", i, g, w)
			}
		}
	})

	t.Run("callees", func(t *testing.T) {
		resp, err := client.Callees(context.Background(), connect.NewRequest(&uiv1.CalleesRequest{Symbol: "Alpha"}))
		if err != nil {
			t.Fatalf("Callees: %v", err)
		}
		eng, closeFn := openIndependentEngine(t)
		defer closeFn()
		want, err := eng.Callees("Alpha", 0)
		if err != nil {
			t.Fatalf("eng.Callees (independent verification path): %v", err)
		}
		if len(want.Callees) == 0 {
			t.Fatal("test fixture assumption broken: eng.Callees(\"Alpha\") returned no callees")
		}
		if resp.Msg.GetSymbol() != want.Symbol {
			t.Fatalf("Callees symbol = %q, want %q", resp.Msg.GetSymbol(), want.Symbol)
		}
		got := resp.Msg.GetCallees()
		if len(got) != len(want.Callees) {
			t.Fatalf("Callees returned %d locations, want %d (independently computed)", len(got), len(want.Callees))
		}
		for i, w := range want.Callees {
			g := got[i]
			if g.GetName() != w.Name || g.GetKind() != w.Kind || g.GetFilePath() != w.FilePath || g.GetStartLine() != w.StartLine {
				t.Fatalf("Callees location[%d] = %+v, want %+v", i, g, w)
			}
		}
	})

	t.Run("impact", func(t *testing.T) {
		resp, err := client.Impact(context.Background(), connect.NewRequest(&uiv1.ImpactRequest{Symbol: "helper"}))
		if err != nil {
			t.Fatalf("Impact: %v", err)
		}
		eng, closeFn := openIndependentEngine(t)
		defer closeFn()
		want, err := eng.Impact("helper", 0)
		if err != nil {
			t.Fatalf("eng.Impact (independent verification path): %v", err)
		}
		if resp.Msg.GetSymbol() != want.Symbol {
			t.Fatalf("Impact symbol = %q, want %q", resp.Msg.GetSymbol(), want.Symbol)
		}
		if resp.Msg.GetDepth() != int32(want.Depth) {
			t.Fatalf("Impact depth = %d, want %d (independently computed)", resp.Msg.GetDepth(), want.Depth)
		}
		if resp.Msg.GetNodeCount() != int32(want.NodeCount) {
			t.Fatalf("Impact node_count = %d, want %d (independently computed)", resp.Msg.GetNodeCount(), want.NodeCount)
		}
		if resp.Msg.GetEdgeCount() != int32(want.EdgeCount) {
			t.Fatalf("Impact edge_count = %d, want %d (independently computed)", resp.Msg.GetEdgeCount(), want.EdgeCount)
		}
		got := resp.Msg.GetAffected()
		if len(got) != len(want.Affected) {
			t.Fatalf("Impact affected has %d locations, want %d (independently computed)", len(got), len(want.Affected))
		}
		for i, w := range want.Affected {
			g := got[i]
			if g.GetName() != w.Name || g.GetKind() != w.Kind || g.GetFilePath() != w.FilePath || g.GetStartLine() != w.StartLine {
				t.Fatalf("Impact affected[%d] = %+v, want %+v", i, g, w)
			}
		}
	})

	t.Run("affected", func(t *testing.T) {
		files := []string{"pkga/pkga.go"}
		resp, err := client.Affected(context.Background(), connect.NewRequest(&uiv1.AffectedRequest{Files: files}))
		if err != nil {
			t.Fatalf("Affected: %v", err)
		}
		eng, closeFn := openIndependentEngine(t)
		defer closeFn()
		want, err := eng.Affected(files, 0)
		if err != nil {
			t.Fatalf("eng.Affected (independent verification path): %v", err)
		}
		if len(resp.Msg.GetFiles()) != len(want.Files) {
			t.Fatalf("Affected files has %d entries, want %d (independently computed)", len(resp.Msg.GetFiles()), len(want.Files))
		}
		for i, f := range want.Files {
			if resp.Msg.GetFiles()[i] != f {
				t.Fatalf("Affected files[%d] = %q, want %q", i, resp.Msg.GetFiles()[i], f)
			}
		}
		got := resp.Msg.GetAffectedTests()
		if len(got) != len(want.AffectedTests) {
			t.Fatalf("Affected affected_tests has %d entries, want %d (independently computed)", len(got), len(want.AffectedTests))
		}
		for i, w := range want.AffectedTests {
			g := got[i]
			if g.GetName() != w.Name || g.GetKind() != w.Kind || g.GetFilePath() != w.FilePath || g.GetStartLine() != w.StartLine {
				t.Fatalf("Affected affected_tests[%d] = %+v, want %+v", i, g, w)
			}
		}
	})

	// depth/limit bounds are the Engine's alone. For Impact/Affected, an
	// explicit depth above the Engine's clamp ceiling (MaxDepth) is
	// silently CLAMPED (validateDepth only rejects a negative depth;
	// clampDepth/clampAffectedDepth then cap anything above MaxDepth) —
	// so this asserts a SUCCESSFUL response whose depth is the Engine's
	// own clamped value, never the caller's raw over-max request. For
	// Callers/Callees, an explicit limit above the Engine's ceiling
	// (MaxLimit) is a DIFFERENT shape: validateLimit REJECTS it outright
	// as ErrInvalidArgument before any traversal runs (the len(locs) >
	// MaxLimit clamp inside Callers/Callees only ever fires for a
	// naturally-oversized RESULT under an in-range request, never for an
	// explicit out-of-range LIMIT argument). Both are the SAME underlying
	// principle this subtest proves: the RPC adds no second, independently
	// -drifting bound at this layer — it reflects EXACTLY what the Engine
	// itself does with the identical argument, whether that is a clamp or
	// a rejection.
	t.Run("limit-above-MaxLimit-is-clamped-by-the-Engine", func(t *testing.T) {
		overDepth := int32(query.MaxDepth) + 5
		resp, err := client.Impact(context.Background(), connect.NewRequest(&uiv1.ImpactRequest{Symbol: "helper", Depth: overDepth}))
		if err != nil {
			t.Fatalf("Impact with depth above MaxDepth returned an error, want the Engine's own clamped success: %v", err)
		}
		if resp.Msg.GetDepth() >= overDepth {
			t.Fatalf("Impact depth = %d, want a value clamped below the requested %d", resp.Msg.GetDepth(), overDepth)
		}
		eng, closeFn := openIndependentEngine(t)
		want, err := eng.Impact("helper", int(overDepth))
		closeFn()
		if err != nil {
			t.Fatalf("eng.Impact (independent verification path): %v", err)
		}
		if resp.Msg.GetDepth() != int32(want.Depth) {
			t.Fatalf("Impact depth = %d, want %d (independently computed Engine clamp)", resp.Msg.GetDepth(), want.Depth)
		}

		overLimit := int32(query.MaxLimit) + 1
		if _, err := client.Callers(context.Background(), connect.NewRequest(&uiv1.CallersRequest{Symbol: "helper", Limit: overLimit})); err == nil {
			t.Fatal("Callers with limit above MaxLimit succeeded over the wire, want the Engine's own rejection reflected unchanged")
		} else if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("Callers with limit above MaxLimit: code = %v, want CodeInvalidArgument", code)
		}
		eng2, closeFn2 := openIndependentEngine(t)
		defer closeFn2()
		if _, err := eng2.Callers("helper", int(overLimit)); err == nil {
			t.Fatal("test assumption broken: independently-computed eng.Callers with the same over-limit argument succeeded")
		}
	})

	t.Run("affected-accepts-a-repeated-file-path", func(t *testing.T) {
		files := []string{"pkga/pkga.go", "pkgb/pkgb.go"}
		resp, err := client.Affected(context.Background(), connect.NewRequest(&uiv1.AffectedRequest{Files: files}))
		if err != nil {
			t.Fatalf("Affected: %v", err)
		}
		eng, closeFn := openIndependentEngine(t)
		defer closeFn()
		want, err := eng.Affected(files, 0)
		if err != nil {
			t.Fatalf("eng.Affected (independent verification path): %v", err)
		}
		if len(resp.Msg.GetFiles()) != len(files) {
			t.Fatalf("Affected files has %d entries, want %d (the repeated request field, echoed back)", len(resp.Msg.GetFiles()), len(files))
		}
		for i, f := range files {
			if resp.Msg.GetFiles()[i] != f {
				t.Fatalf("Affected files[%d] = %q, want %q", i, resp.Msg.GetFiles()[i], f)
			}
		}
		if len(resp.Msg.GetAffectedTests()) != len(want.AffectedTests) {
			t.Fatalf("Affected affected_tests has %d entries, want %d (independently computed for the same repeated file set)", len(resp.Msg.GetAffectedTests()), len(want.AffectedTests))
		}
	})

	t.Run("unknown-symbol-is-CodeNotFound-not-CodeInternal", func(t *testing.T) {
		_, err := client.Callers(context.Background(), connect.NewRequest(&uiv1.CallersRequest{Symbol: "NoSuchSymbolXYZ123"}))
		if err == nil {
			t.Fatal("Callers for an unknown symbol succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeNotFound {
			t.Fatalf("Callers for an unknown symbol: code = %v, want CodeNotFound (got error %v)", code, err)
		}
	})
}

// buildOverloadedFixture indexes a synthetic module with n distinct
// packages, each declaring exactly one exported function literally named
// funcName plus its own uniquely-named unexported helper — Go tolerates
// the same function name across separate packages (unlike within one
// package), giving GetNodeDetail a real overloaded symbol with n
// distinct schema.Node definitions, one per package/file. Package i's
// function calls its own helperN so each candidate's Calls is
// independently distinguishable (never a shared or collapsed value); a
// single external "caller" package calls ONLY package 0's function (a
// real cross-package qualified call, the same pattern
// internal/indexer/testdata/gofixture/pkgb/pkgb.go already exercises for
// pkga.Alpha), so CalledBy is non-empty for exactly one candidate and
// empty for the rest — proving a wire consumer cannot observe a shared or
// collapsed per-candidate value (D-02).
func buildOverloadedFixture(t *testing.T, funcName string, n int) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/multidef\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for i := 0; i < n; i++ {
		pkgDir := filepath.Join(dir, fmt.Sprintf("pkg%d", i))
		if err := os.MkdirAll(pkgDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pkgDir, err)
		}
		src := fmt.Sprintf("package pkg%d\n\nfunc helper%d() {}\n\nfunc %s() {\n\thelper%d()\n}\n", i, i, funcName, i)
		if err := os.WriteFile(filepath.Join(pkgDir, fmt.Sprintf("pkg%d.go", i)), []byte(src), 0o644); err != nil {
			t.Fatalf("write pkg%d.go: %v", i, err)
		}
	}
	callerDir := filepath.Join(dir, "caller")
	if err := os.MkdirAll(callerDir, 0o755); err != nil {
		t.Fatalf("mkdir caller: %v", err)
	}
	callerSrc := fmt.Sprintf("package caller\n\nimport \"example.com/multidef/pkg0\"\n\nfunc InvokeFirst() {\n\tpkg0.%s()\n}\n", funcName)
	if err := os.WriteFile(filepath.Join(callerDir, "caller.go"), []byte(callerSrc), 0o644); err != nil {
		t.Fatalf("write caller.go: %v", err)
	}
	indexGofixture(t, dir)
	return dir
}

// TestUIServiceNodeDetailCoversAllThreeModes proves GetNodeDetail answers
// all three internal/query.NodeDetail shapes over the wire (D-02): file
// mode returns the path, single-def mode returns a node plus calls plus
// called-by, and multi-def mode returns the symbol plus a populated
// candidate list — with every field belonging to the OTHER two modes left
// at its zero value in each case.
func TestUIServiceNodeDetailCoversAllThreeModes(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: "pkga/pkga.go"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(file): %v", err)
		}
		if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_FILE {
			t.Fatalf("GetNodeDetail(file): mode = %v, want NODE_DETAIL_MODE_FILE", resp.Msg.GetMode())
		}
		if resp.Msg.GetPath() != "pkga/pkga.go" {
			t.Fatalf("GetNodeDetail(file): path = %q, want %q", resp.Msg.GetPath(), "pkga/pkga.go")
		}
		if resp.Msg.GetNode() != nil || len(resp.Msg.GetCalls()) != 0 || len(resp.Msg.GetCalledBy()) != 0 {
			t.Fatal("GetNodeDetail(file): single-def fields populated, want all empty for file mode")
		}
		if resp.Msg.GetSymbol() != "" || len(resp.Msg.GetDefinitions()) != 0 || resp.Msg.GetTotalCandidates() != 0 {
			t.Fatal("GetNodeDetail(file): multi-def fields populated, want all empty for file mode")
		}
	})

	t.Run("single-def", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Alpha"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(Alpha): %v", err)
		}
		if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_SINGLE_DEF {
			t.Fatalf("GetNodeDetail(Alpha): mode = %v, want NODE_DETAIL_MODE_SINGLE_DEF", resp.Msg.GetMode())
		}
		if resp.Msg.GetNode() == nil || resp.Msg.GetNode().GetName() != "Alpha" {
			t.Fatalf("GetNodeDetail(Alpha): node = %v, want a populated Alpha node", resp.Msg.GetNode())
		}
		if len(resp.Msg.GetCalls()) == 0 {
			t.Fatal("GetNodeDetail(Alpha): calls is empty, want Alpha's call to helper")
		}
		if len(resp.Msg.GetCalledBy()) == 0 {
			t.Fatal("GetNodeDetail(Alpha): called_by is empty, want pkgb.Run's call to Alpha")
		}
		if resp.Msg.GetPath() != "" {
			t.Fatalf("GetNodeDetail(Alpha): path = %q, want empty for single-def mode", resp.Msg.GetPath())
		}
		if resp.Msg.GetSymbol() != "" || len(resp.Msg.GetDefinitions()) != 0 {
			t.Fatal("GetNodeDetail(Alpha): multi-def fields populated, want empty for single-def mode")
		}
	})

	t.Run("multi-def", func(t *testing.T) {
		dir := buildOverloadedFixture(t, "Overload", 2)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Overload"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(Overload): %v", err)
		}
		if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_MULTI_DEF {
			t.Fatalf("GetNodeDetail(Overload): mode = %v, want NODE_DETAIL_MODE_MULTI_DEF", resp.Msg.GetMode())
		}
		if resp.Msg.GetSymbol() != "Overload" {
			t.Fatalf("GetNodeDetail(Overload): symbol = %q, want %q", resp.Msg.GetSymbol(), "Overload")
		}
		if len(resp.Msg.GetDefinitions()) != 2 {
			t.Fatalf("GetNodeDetail(Overload): got %d definitions, want 2", len(resp.Msg.GetDefinitions()))
		}
		if resp.Msg.GetTotalCandidates() != 2 {
			t.Fatalf("GetNodeDetail(Overload): total_candidates = %d, want 2", resp.Msg.GetTotalCandidates())
		}
		if resp.Msg.GetPath() != "" || resp.Msg.GetNode() != nil {
			t.Fatal("GetNodeDetail(Overload): file/single-def fields populated, want empty for multi-def mode")
		}
	})
}

// TestUIServiceNodeDetailGoesThroughEngineBuilder proves GetNodeDetail's
// calls/called-by sets equal (*query.Engine).NodeDetail's for the SAME
// symbol, computed independently in this test — for both the
// single-definition case and, per-candidate, the multi-definition case
// (D-01): the UI is a third consumer of the one gather path, never a
// second implementation.
func TestUIServiceNodeDetailGoesThroughEngineBuilder(t *testing.T) {
	// Single-definition case.
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Alpha"}))
	if err != nil {
		t.Fatalf("GetNodeDetail(Alpha): %v", err)
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	want, err := eng.NodeDetail("Alpha", "", nil)
	closer.Close()
	if err != nil {
		t.Fatalf("eng.NodeDetail(Alpha) (independent verification path): %v", err)
	}
	if want.Mode != query.NodeDetailModeSingleDef {
		t.Fatal("test fixture assumption broken: eng.NodeDetail(Alpha) is not single-def")
	}
	if len(resp.Msg.GetCalls()) != len(want.Definition.Calls) {
		t.Fatalf("GetNodeDetail(Alpha) calls has %d entries, want %d (independently computed)", len(resp.Msg.GetCalls()), len(want.Definition.Calls))
	}
	for i, w := range want.Definition.Calls {
		if resp.Msg.GetCalls()[i].GetName() != w.Name {
			t.Fatalf("GetNodeDetail(Alpha) calls[%d] = %q, want %q", i, resp.Msg.GetCalls()[i].GetName(), w.Name)
		}
	}
	if len(resp.Msg.GetCalledBy()) != len(want.Definition.CalledBy) {
		t.Fatalf("GetNodeDetail(Alpha) called_by has %d entries, want %d (independently computed)", len(resp.Msg.GetCalledBy()), len(want.Definition.CalledBy))
	}
	for i, w := range want.Definition.CalledBy {
		if resp.Msg.GetCalledBy()[i].GetName() != w.Name {
			t.Fatalf("GetNodeDetail(Alpha) called_by[%d] = %q, want %q", i, resp.Msg.GetCalledBy()[i].GetName(), w.Name)
		}
	}

	// Multi-definition case: each returned candidate's calls/called-by
	// must equal THAT candidate's own per-candidate lookup — matched by
	// node id, never by position, since match order is not this test's
	// claim.
	multiDir := buildOverloadedFixture(t, "Overload", 2)
	multiSrv := startedServer(t, multiDir)
	multiClient := uiv1connect.NewUIServiceClient(http.DefaultClient, multiSrv.URL())

	multiResp, err := multiClient.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Overload"}))
	if err != nil {
		t.Fatalf("GetNodeDetail(Overload): %v", err)
	}

	multiEng, multiCloser, err := query.OpenAt(multiDir)
	if err != nil {
		t.Fatalf("query.OpenAt (multi-def independent verification path): %v", err)
	}
	defer multiCloser.Close()
	wantMulti, err := multiEng.NodeDetail("Overload", "", nil)
	if err != nil {
		t.Fatalf("eng.NodeDetail(Overload) (independent verification path): %v", err)
	}
	if wantMulti.Mode != query.NodeDetailModeMultiDef {
		t.Fatal("test fixture assumption broken: eng.NodeDetail(Overload) is not multi-def")
	}
	if len(multiResp.Msg.GetDefinitions()) != len(wantMulti.Multi.Matches) {
		t.Fatalf("GetNodeDetail(Overload) returned %d definitions, want %d (independently computed)", len(multiResp.Msg.GetDefinitions()), len(wantMulti.Multi.Matches))
	}

	byID := make(map[string]*uiv1.NodeDefinition, len(multiResp.Msg.GetDefinitions()))
	for _, def := range multiResp.Msg.GetDefinitions() {
		byID[def.GetNode().GetId()] = def
	}

	var sawNonEmptyCalledBy, sawEmptyCalledBy bool
	for _, match := range wantMulti.Multi.Matches {
		gotDef, ok := byID[match.Id]
		if !ok {
			t.Fatalf("GetNodeDetail(Overload) definitions has no entry for independently-computed candidate id %q", match.Id)
		}
		wantDD, err := wantMulti.Multi.Definition(match)
		if err != nil {
			t.Fatalf("wantMulti.Multi.Definition(%s) (independent verification path): %v", match.Id, err)
		}
		if len(gotDef.GetCalls()) != len(wantDD.Calls) {
			t.Fatalf("GetNodeDetail(Overload) definition[id=%s].calls has %d entries, want %d (that candidate's OWN per-candidate lookup)", match.Id, len(gotDef.GetCalls()), len(wantDD.Calls))
		}
		for j, w := range wantDD.Calls {
			if gotDef.GetCalls()[j].GetName() != w.Name {
				t.Fatalf("GetNodeDetail(Overload) definition[id=%s].calls[%d] = %q, want %q", match.Id, j, gotDef.GetCalls()[j].GetName(), w.Name)
			}
		}
		if len(gotDef.GetCalledBy()) != len(wantDD.CalledBy) {
			t.Fatalf("GetNodeDetail(Overload) definition[id=%s].called_by has %d entries, want %d (that candidate's OWN per-candidate lookup)", match.Id, len(gotDef.GetCalledBy()), len(wantDD.CalledBy))
		}
		for j, w := range wantDD.CalledBy {
			if gotDef.GetCalledBy()[j].GetName() != w.Name {
				t.Fatalf("GetNodeDetail(Overload) definition[id=%s].called_by[%d] = %q, want %q", match.Id, j, gotDef.GetCalledBy()[j].GetName(), w.Name)
			}
		}
		if len(gotDef.GetCalledBy()) > 0 {
			sawNonEmptyCalledBy = true
		} else {
			sawEmptyCalledBy = true
		}
	}
	if !sawNonEmptyCalledBy || !sawEmptyCalledBy {
		t.Fatal("test fixture assumption broken: want at least one Overload candidate with non-empty called_by and at least one with empty called_by — otherwise a mapper collapsing every candidate to the same value could pass this test vacuously")
	}
}

// TestUIServiceNodeDetailCapsCandidatesAndReportsTheTotal proves a symbol
// with more definitions than uiMultiDefCap returns exactly uiMultiDefCap
// candidates carrying gathered detail, the remaining candidates listed
// without detail, and a total-candidate count equal to the real number
// of matches.
func TestUIServiceNodeDetailCapsCandidatesAndReportsTheTotal(t *testing.T) {
	const total = uiMultiDefCap + 2
	dir := buildOverloadedFixture(t, "Big", total)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Big"}))
	if err != nil {
		t.Fatalf("GetNodeDetail(Big): %v", err)
	}
	if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_MULTI_DEF {
		t.Fatalf("GetNodeDetail(Big): mode = %v, want NODE_DETAIL_MODE_MULTI_DEF", resp.Msg.GetMode())
	}
	if resp.Msg.GetTotalCandidates() != int32(total) {
		t.Fatalf("GetNodeDetail(Big): total_candidates = %d, want %d (the real match count)", resp.Msg.GetTotalCandidates(), total)
	}
	if len(resp.Msg.GetDefinitions()) != total {
		t.Fatalf("GetNodeDetail(Big): got %d definitions, want %d (every candidate listed)", len(resp.Msg.GetDefinitions()), total)
	}

	var gathered, listed int
	for _, def := range resp.Msg.GetDefinitions() {
		if def.GetNode() == nil {
			t.Fatal("GetNodeDetail(Big): a definition has no node record")
		}
		if def.GetDetailGathered() {
			gathered++
		} else {
			listed++
			if len(def.GetCalls()) != 0 || len(def.GetCalledBy()) != 0 {
				t.Fatal("GetNodeDetail(Big): a listed (non-gathered) definition carries calls/called_by, want both empty")
			}
		}
	}
	if gathered != uiMultiDefCap {
		t.Fatalf("GetNodeDetail(Big): %d definitions carry gathered detail, want exactly uiMultiDefCap (%d)", gathered, uiMultiDefCap)
	}
	if listed != total-uiMultiDefCap {
		t.Fatalf("GetNodeDetail(Big): %d definitions listed without detail, want %d", listed, total-uiMultiDefCap)
	}
}

// TestUIServiceNodeDetailErrorClasses proves GetNodeDetail classifies
// every caller-reachable rejection with the mapped Connect code rather
// than an unclassified default: a not-found symbol, a request with
// neither symbol nor file, and a per-candidate lookup that fails within
// the cap (an unreadable candidate file) — the last case fails the WHOLE
// RPC rather than returning a partially-populated response.
func TestUIServiceNodeDetailErrorClasses(t *testing.T) {
	t.Run("symbol-not-found", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "NoSuchSymbolXYZ123"}))
		if err == nil {
			t.Fatal("GetNodeDetail for an unknown symbol succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeNotFound {
			t.Fatalf("GetNodeDetail for an unknown symbol: code = %v, want CodeNotFound", code)
		}
	})

	t.Run("neither-symbol-nor-file", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{}))
		if err == nil {
			t.Fatal("GetNodeDetail with neither symbol nor file succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Fatalf("GetNodeDetail with neither symbol nor file: code = %v, want CodeInvalidArgument", code)
		}
	})

	t.Run("unreadable-candidate-within-the-cap", func(t *testing.T) {
		dir := buildOverloadedFixture(t, "Broken", 2)
		// Corrupt EVERY candidate's file after indexing: the Pebble store
		// already has the schema.Node records, and a candidate's FilePath
		// is read fresh from disk at RPC time (never from the store), so
		// replacing the file with a directory of the same name makes ANY
		// within-cap candidate's per-candidate lookup fail deterministically
		// regardless of matches order — mirroring 01-04's directory-backed
		// unreadable-candidate technique
		// (TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates).
		for i := 0; i < 2; i++ {
			p := filepath.Join(dir, fmt.Sprintf("pkg%d", i), fmt.Sprintf("pkg%d.go", i))
			if err := os.Remove(p); err != nil {
				t.Fatalf("remove %s: %v", p, err)
			}
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatalf("mkdir %s (in place of the file): %v", p, err)
			}
		}

		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		_, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Broken"}))
		if err == nil {
			t.Fatal("GetNodeDetail with every within-cap candidate's file replaced by a directory succeeded, want an error")
		}
		if code := connect.CodeOf(err); code != connect.CodeInternal {
			t.Fatalf("GetNodeDetail with an unreadable within-cap candidate: code = %v, want CodeInternal (an unclassified read failure, not a resolveSourcePath rejection)", code)
		}
	})
}

// copyBehavioralFixture mirrors internal/query/explore_test.go's
// copyBehavioralFixture byte-for-byte (unexported in package query,
// reproduced here per this repo's existing "test scaffolding" convention
// — see copyGofixture's own comment above): it copies the behavioral
// corpus source tree (corpus/behavioral/src, D-03) into a fresh
// t.TempDir(), skipping any committed .codegraph/ directory since this
// test builds a fresh index via indexGofixture (indexer.Run) against the
// CURRENT extractors/schema. Explore's wire tests need a REAL multi-file
// corpus — gofixture's handful of tiny files do not reliably produce
// several ranked file groups for a real query.
func copyBehavioralFixture(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs(filepath.Join("..", "..", "corpus", "behavioral", "src"))
	if err != nil {
		t.Fatalf("resolve behavioral corpus fixture path: %v", err)
	}
	dst := t.TempDir()
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".codegraph" {
			return filepath.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy behavioral fixture: %v", err)
	}
	return dst
}

// TestUIServiceExploreGroupsCarryTheirOwnSources proves the wire mapping
// preserves ExploreResult's three per-group/per-blast associations
// (D-01, D-02) for a query producing several groups: each group's path is
// a valid lookup key into the independently-computed
// ExploreResult.Sources map (the exact keying plan 01-10's forthcoming
// per-group source field will use — see ui.proto's header comment); the
// skeletonized flag is set for exactly the group paths present in
// ExploreResult.SkeletonFiles; and blast entries map one-to-one onto
// ExploreResult.Blasts.
func TestUIServiceExploreGroupsCarryTheirOwnSources(t *testing.T) {
	dir := copyBehavioralFixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	const query1 = "account balance"

	resp, err := client.Explore(context.Background(), connect.NewRequest(&uiv1.ExploreRequest{Query: query1}))
	if err != nil {
		t.Fatalf("Explore(%q): %v", query1, err)
	}
	if resp.Msg.GetEmpty() {
		t.Fatalf("Explore(%q): empty = true, want a populated result (test fixture assumption)", query1)
	}
	if len(resp.Msg.GetGroups()) < 3 {
		t.Fatalf("Explore(%q) returned %d groups, want at least 3 (test fixture assumption)", query1, len(resp.Msg.GetGroups()))
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.ExploreDetail(query1, 0)
	if err != nil {
		t.Fatalf("eng.ExploreDetail(%q) (independent verification path): %v", query1, err)
	}
	if len(want.Groups) != len(resp.Msg.GetGroups()) {
		t.Fatalf("Explore(%q) returned %d groups, want %d (independently computed)", query1, len(resp.Msg.GetGroups()), len(want.Groups))
	}

	t.Run("source-matches-group-path", func(t *testing.T) {
		for _, g := range resp.Msg.GetGroups() {
			src, ok := want.Sources[g.GetPath()]
			if !ok || len(src) == 0 {
				t.Fatalf("wire group path %q has no corresponding non-empty entry in the independently-computed ExploreResult.Sources — the wire path is not a valid source-lookup key", g.GetPath())
			}
		}
	})

	t.Run("skeletonized-flag-set-exactly-for-SkeletonFiles", func(t *testing.T) {
		for _, g := range resp.Msg.GetGroups() {
			wantSkeleton := want.SkeletonFiles[g.GetPath()]
			if g.GetSkeletonized() != wantSkeleton {
				t.Fatalf("group %q: skeletonized = %v, want %v (ExploreResult.SkeletonFiles[%q])", g.GetPath(), g.GetSkeletonized(), wantSkeleton, g.GetPath())
			}
		}
	})

	t.Run("blast-entries-map-one-to-one", func(t *testing.T) {
		gotBlasts := resp.Msg.GetBlasts()
		if len(gotBlasts) != len(want.Blasts) {
			t.Fatalf("Explore(%q) returned %d blast entries, want %d (independently computed)", query1, len(gotBlasts), len(want.Blasts))
		}
		for i, w := range want.Blasts {
			g := gotBlasts[i]
			if g.GetSymbol().GetName() != w.Symbol.Name {
				t.Fatalf("blast[%d].symbol.name = %q, want %q", i, g.GetSymbol().GetName(), w.Symbol.Name)
			}
			if g.GetCallerCount() != int32(w.CallerCount) {
				t.Fatalf("blast[%d].caller_count = %d, want %d", i, g.GetCallerCount(), w.CallerCount)
			}
			if len(g.GetTestFiles()) != len(w.TestFiles) {
				t.Fatalf("blast[%d].test_files has %d entries, want %d", i, len(g.GetTestFiles()), len(w.TestFiles))
			}
			for j, tf := range w.TestFiles {
				if g.GetTestFiles()[j] != tf {
					t.Fatalf("blast[%d].test_files[%d] = %q, want %q", i, j, g.GetTestFiles()[j], tf)
				}
			}
		}
	})
}

// TestUIServiceExploreZeroMatchIsNotAnError proves a query matching
// nothing returns a SUCCESSFUL response with the empty marker set and no
// groups — never connect.CodeNotFound and never connect.CodeInternal
// (D-02): ExploreResult models the empty case explicitly for exactly
// this reason.
func TestUIServiceExploreZeroMatchIsNotAnError(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.Explore(context.Background(), connect.NewRequest(&uiv1.ExploreRequest{Query: "zzzznonexistentquerytermxyz987"}))
	if err != nil {
		t.Fatalf("Explore with a query matching nothing returned an error, want a successful empty response: %v", err)
	}
	if !resp.Msg.GetEmpty() {
		t.Fatal("Explore with a query matching nothing: empty = false, want true")
	}
	if len(resp.Msg.GetGroups()) != 0 {
		t.Fatalf("Explore with a query matching nothing: got %d groups, want 0", len(resp.Msg.GetGroups()))
	}
}

// TestUIServiceExploreRejectsEmptyQuery proves an empty query is rejected
// as connect.CodeInvalidArgument, reflecting Engine.ExploreDetail's own
// rejection rather than a second, independently-drifting copy of the
// rule at this layer.
func TestUIServiceExploreRejectsEmptyQuery(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err := client.Explore(context.Background(), connect.NewRequest(&uiv1.ExploreRequest{Query: ""}))
	if err == nil {
		t.Fatal("Explore with an empty query succeeded, want an error")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("Explore with an empty query: code = %v, want CodeInvalidArgument", code)
	}
}

// TestUIServiceAffectedEnforcesTheFileCountCap proves CR-03's contract:
// the Affected RPC enforces query.ValidateAffectedFiles on the
// wire-supplied files slice, exactly as internal/cli/affected.go's
// collectAffectedFiles does before it calls Engine.Affected.
//
// The cap is the Engine's documented DoS ceiling (MaxAffectedFiles,
// added under CR-01/T-03-02-DoS) and ValidateAffectedFiles is exported
// precisely so a caller can enforce it BEFORE Engine.Affected is ever
// reached — but Engine.Affected itself validates only depth, so a new
// surface that forgets the pre-check inherits no bound at all. The wire
// is exactly such a surface: the only remaining limit on a `repeated
// string` is transportReadMaxBytes (1 MiB), which at a few bytes per
// short path admits well over an order of magnitude more entries than
// the documented ceiling.
//
// Both sides of the boundary are asserted, so the test cannot pass
// vacuously: MaxAffectedFiles entries must still be ACCEPTED (a
// successful response), and MaxAffectedFiles+1 must be REJECTED with
// CodeInvalidArgument. Removing the handler's check turns the
// over-the-cap subtest RED — without it the request succeeds.
func TestUIServiceAffectedEnforcesTheFileCountCap(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	// Short, distinct, in-repo-shaped paths: the point is the COUNT, not
	// whether any of them resolves to a real indexed file.
	buildFiles := func(n int) []string {
		files := make([]string, n)
		for i := range files {
			files[i] = fmt.Sprintf("pkga/f%d.go", i)
		}
		return files
	}

	t.Run("at-the-cap-is-accepted", func(t *testing.T) {
		_, err := client.Affected(context.Background(), connect.NewRequest(&uiv1.AffectedRequest{
			Files: buildFiles(query.MaxAffectedFiles),
		}))
		if err != nil {
			t.Fatalf("Affected with exactly MaxAffectedFiles (%d) entries: %v, want a successful response — the cap rejects only what is ABOVE it", query.MaxAffectedFiles, err)
		}
	})

	t.Run("over-the-cap-is-rejected", func(t *testing.T) {
		_, err := client.Affected(context.Background(), connect.NewRequest(&uiv1.AffectedRequest{
			Files: buildFiles(query.MaxAffectedFiles + 1),
		}))
		if err == nil {
			t.Fatalf("Affected with MaxAffectedFiles+1 (%d) entries succeeded, want CodeInvalidArgument — the documented DoS ceiling is not enforced at this surface", query.MaxAffectedFiles+1)
		}
		if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
			t.Fatalf("Affected over the cap: code = %v (%v), want CodeInvalidArgument", got, err)
		}
		if !strings.Contains(err.Error(), "exceeds maximum") {
			t.Fatalf("Affected over the cap: message = %q, want it to carry ValidateAffectedFiles's own %q wording", err.Error(), "exceeds maximum")
		}
	})
}

// TestWithEngineRefusesAnAlreadyCancelledRequest proves WR-02's
// pre-flight gate: withEngine's ctx parameter is actually consulted, so
// a request whose client has already gone away is refused before the
// store is opened and before any Engine work begins.
//
// Both directions are asserted so the test cannot pass vacuously:
//
//   - cancelled ctx: fn must NOT run, and the error must be
//     CodeCanceled — not CodeInternal, which is what routing a
//     context.Canceled through mapEngineError's default arm would
//     produce for an ordinary browser-tab close;
//   - expired deadline: CodeDeadlineExceeded, its own distinct code;
//   - live ctx (the positive control): fn MUST run and its result must
//     propagate, proving the gate rejects only what it should.
//
// Removing the ctx.Err() check turns the first two subtests RED (fn
// runs and the call succeeds).
func TestWithEngineRefusesAnAlreadyCancelledRequest(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ran := false
		err := withEngine(ctx, dir, func(*query.Engine) error {
			ran = true
			return nil
		})
		if err == nil {
			t.Fatal("withEngine with an already-cancelled ctx returned nil, want CodeCanceled")
		}
		if got := connect.CodeOf(err); got != connect.CodeCanceled {
			t.Fatalf("withEngine with a cancelled ctx: code = %v (%v), want CodeCanceled", got, err)
		}
		if ran {
			t.Fatal("withEngine ran fn for an already-cancelled request — the pre-flight gate must refuse before any Engine work begins")
		}
	})

	t.Run("deadline-exceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		ran := false
		err := withEngine(ctx, dir, func(*query.Engine) error {
			ran = true
			return nil
		})
		if got := connect.CodeOf(err); got != connect.CodeDeadlineExceeded {
			t.Fatalf("withEngine with an expired deadline: code = %v (%v), want CodeDeadlineExceeded", got, err)
		}
		if ran {
			t.Fatal("withEngine ran fn for an already-expired request")
		}
	})

	t.Run("live-ctx-still-runs", func(t *testing.T) {
		ran := false
		err := withEngine(context.Background(), dir, func(eng *query.Engine) error {
			ran = true
			return nil
		})
		if err != nil {
			t.Fatalf("withEngine with a live ctx: %v, want success", err)
		}
		if !ran {
			t.Fatal("withEngine did not run fn for a live request — the pre-flight gate must reject only cancelled/expired contexts")
		}
	})
}
