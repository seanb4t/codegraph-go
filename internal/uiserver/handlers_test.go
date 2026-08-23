package uiserver

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

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
	go srv.Serve(ctx)
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
