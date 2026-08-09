package mcp

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// copyFixture copies internal/indexer/testdata/gofixture into a fresh
// t.TempDir(), mirroring internal/query/engine_test.go's helper of the
// same name (03-PATTERNS.md §"Test scaffolding") so this package's tests
// run against a normal directory with its own go.mod rather than
// mutating the checked-in testdata tree.
func copyFixture(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "indexer", "testdata", "gofixture"))
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
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
		t.Fatalf("copy fixture: %v", err)
	}
	return dst
}

// indexFixture builds a real graph from the fixture rooted at dir into
// <dir>/.codegraph/store via indexer.Run, mirroring
// internal/query/engine_test.go's helper of the same name — a real
// Pebble-backed store, not a mock, so the handler-delegation test
// exercises the actual engine read path.
func indexFixture(t *testing.T, dir string) {
	t.Helper()

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

// newTestSession builds an in-memory client/server session pair for s
// (Phase 2, RESEARCH Testing Architecture): mcp.NewInMemoryTransports()
// returns two *InMemoryTransport halves; s.Run is started on one in a
// goroutine, and a client connects to the other via Connect, which
// performs the MCP initialize handshake itself — go-sdk's Client has no
// separate, repeatable Initialize call the way mark3labs' client did
// (initClient no longer exists; this is the PROVEN-ABSENT finding
// 02-RESEARCH.md Q1 asked to be confirmed rather than assumed: no
// ServerOptions field or client method lets a caller inject a
// ProtocolVersion — the repo-owned ProtocolVersion constant keeps its
// role as the asserted pin). Every other test file in this package uses
// this one helper rather than repeating the construction.
//
// The returned cleanup func closes the session; callers are responsible
// for calling it (directly or via defer).
func newTestSession(t *testing.T, s *mcp.Server) (*mcp.ClientSession, func()) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-mcp-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}

	return session, func() { _ = session.Close() }
}

// listToolNames connects to s via newTestSession and returns the sorted
// set of registered tool names.
func listToolNames(t *testing.T, s *mcp.Server) []string {
	t.Helper()

	session, cleanup := newTestSession(t, s)
	defer cleanup()

	result, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	sort.Strings(names)
	return names
}

// allEightToolNames returns every tool name the default surface registers,
// sorted — derived from exploreTool()/companionTool() via allToolNames()
// rather than re-typed, so adding an eighth companion extends every
// expectation below automatically instead of silently passing.
func allEightToolNames(t *testing.T) []string {
	t.Helper()
	names := append([]string(nil), allToolNames()...)
	sort.Strings(names)
	return names
}

// TestDefaultToolVisibility pins the DEFAULT surface: CODEGRAPH_MCP_TOOLS
// unset, an index present, all eight tools registered.
//
// This expectation was moved deliberately. It previously asserted exactly
// [codegraph_explore] — the pre-inversion opt-in default — and passing it
// through BuildServer with a hand-built empty map meant the test could not
// see the default at all: it pinned BuildServer's behavior for one
// particular map, while the actual default lived in serve.go's os.Getenv
// call, untested. Routing through ResolveCompanions is the point of this
// test, not a stylistic detail: this is the exact path production takes,
// and the exact case the reported "the mcp server is only showing one tool"
// bug landed on.
func TestDefaultToolVisibility(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	// present=false is what os.LookupEnv reports for an unset variable.
	companions, unknown := ResolveCompanions("", false)
	if len(unknown) != 0 {
		t.Fatalf("ResolveCompanions with the variable unset reported unknown names %v, want none", unknown)
	}

	s := BuildServer(true, companions, dir, dir)

	got := listToolNames(t, s)
	want := allEightToolNames(t)
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v (default is every tool when CODEGRAPH_MCP_TOOLS is unset)", got, want)
	}
}

// TestToolFilterNarrowsToNamedCompanions is the former TestAllowlist,
// retargeted at narrowing semantics: a SET CODEGRAPH_MCP_TOOLS removes
// every companion it does not name, rather than adding every companion it
// does. The observable set for "node,status" is identical under both
// contracts — which is exactly why this test alone cannot detect the
// inversion, and why TestDefaultToolVisibility and
// TestEmptyToolFilterNarrowsToExploreOnly are the two that can.
func TestToolFilterNarrowsToNamedCompanions(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, unknown := ResolveCompanions("node,status,bogus", true)

	if !companions["node"] || !companions["status"] {
		t.Fatalf("ResolveCompanions selected = %v, want node+status set", companions)
	}
	if companions["bogus"] {
		t.Fatalf("ResolveCompanions selected contains unknown name %q", "bogus")
	}
	if len(unknown) != 1 || unknown[0] != "bogus" {
		t.Fatalf("ResolveCompanions unknown = %v, want [bogus]", unknown)
	}

	var stderr bytes.Buffer
	WarnToolFilterTo(&stderr, unknown, companions)
	if !strings.Contains(stderr.String(), "bogus") {
		t.Fatalf("WarnToolFilterTo did not mention %q, got %q", "bogus", stderr.String())
	}
	// The consequence half: naming what was ignored without naming what
	// survived is what makes an all-typo'd value present as "the server is
	// broken" rather than "I narrowed it too far". See WarnToolFilterTo.
	if !strings.Contains(stderr.String(), "2 of 7") {
		t.Fatalf("WarnToolFilterTo did not state the resulting surface (want %q in the output), got %q", "2 of 7", stderr.String())
	}

	s := BuildServer(true, companions, dir, dir)
	got := listToolNames(t, s)
	want := []string{"codegraph_explore", "codegraph_node", "codegraph_status"}
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}
}

// TestEmptyToolFilterNarrowsToExploreOnly is the boundary neighbour the
// inversion creates and the one case os.Getenv structurally cannot express:
// a variable SET to the empty string narrows to codegraph_explore alone,
// while the same empty VALUE with the variable unset registers all eight.
// Both sub-cases run here, in one test, because the property being pinned
// is the DIFFERENCE between them — asserting either alone would pass
// against an implementation that ignored `present` entirely.
func TestEmptyToolFilterNarrowsToExploreOnly(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	setEmpty, unknown := ResolveCompanions("", true)
	if len(unknown) != 0 {
		t.Fatalf("ResolveCompanions(\"\", true) reported unknown names %v, want none", unknown)
	}
	if len(setEmpty) != 0 {
		t.Fatalf("ResolveCompanions(\"\", true) selected %v, want no companions", setEmpty)
	}

	got := listToolNames(t, BuildServer(true, setEmpty, dir, dir))
	want := []string{"codegraph_explore"}
	if !equalStrings(got, want) {
		t.Fatalf("CODEGRAPH_MCP_TOOLS set to the empty string registered %v, want %v", got, want)
	}

	unset, _ := ResolveCompanions("", false)
	gotUnset := listToolNames(t, BuildServer(true, unset, dir, dir))
	if equalStrings(gotUnset, want) {
		t.Fatalf("an UNSET CODEGRAPH_MCP_TOOLS registered the same set as one SET to the empty string (%v) — the present/absent distinction is being ignored, which is exactly the os.Getenv collapse this contract depends on avoiding", gotUnset)
	}
}

func TestNoIndexZeroTools(t *testing.T) {
	dir := t.TempDir()

	s := BuildServer(false, map[string]bool{"node": true, "status": true}, dir, dir)

	got := listToolNames(t, s)
	if len(got) != 0 {
		t.Fatalf("registered tools = %v, want none (MCP-03: no .codegraph/ means zero tools)", got)
	}
}

// TestExploreHandlerDelegatesToEngine proves D-08b: the codegraph_explore
// tool call resolves its path/query args, opens a fresh engine snapshot
// via query.OpenAt, and returns Engine.Explore's markdown — not a
// second, MCP-only rendering path.
func TestExploreHandlerDelegatesToEngine(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{}, dir, dir)

	result := callTool(t, s, "codegraph_explore", map[string]any{
		"query": "main",
		"path":  dir,
	})
	if result.IsError {
		t.Fatalf("codegraph_explore returned an error result: %+v", result)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "**Exploration:") {
		t.Fatalf("codegraph_explore output missing the Engine.Explore markdown header, got: %q", text)
	}
}

// TestOpenEnginePathConfinedToRepoRoot pins CR-02: a client-supplied
// "path" argument that resolves outside the server's configured repo
// root must be rejected with an MCP tool error, never opened as an
// engine — even when that outside path is itself a validly-indexed
// .codegraph/ project (proving this is a trust-boundary confinement
// check, not just an "index not found" failure).
func TestOpenEnginePathConfinedToRepoRoot(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	outside := copyFixture(t)
	indexFixture(t, outside)

	s := BuildServer(true, map[string]bool{"status": true}, dir, dir)

	result := callTool(t, s, "codegraph_status", map[string]any{"path": outside})
	if !result.IsError {
		t.Fatal("codegraph_status with a path outside the server's repo root: expected an error result, got success")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "outside") {
		t.Fatalf("codegraph_status error message = %q, want it to explain the path is outside the repo root", text)
	}
}

// TestConfinementAnchoredOnRepoRootNotStartPath is WR-02's required test
// (02-REVIEW-2.md): CR-01 split openEngine's single confinement parameter
// into two adjacent, same-typed strings (defaultPath/startPath and
// repoPath). TestOpenEnginePathConfinedToRepoRoot above builds its server
// with startPath == repoPath (BuildServer(..., dir, dir)) — a degenerate
// configuration in which the two anchors are indistinguishable, so that
// test passes identically whichever one confineToRepoRoot actually
// anchors on. This test runs the SAME confinement assertion in the
// configuration production actually uses (startPath != repoPath, the
// linked-worktree shape serve.go produces via serveServerPaths), with a
// path chosen specifically to distinguish the two anchors: INSIDE repoPath
// (main) but OUTSIDE startPath (wt).
func TestConfinementAnchoredOnRepoRootNotStartPath(t *testing.T) {
	wt, main := mcpWorktreeMismatchFixture(t) // startPath=wt, repoPath=main

	s := BuildServer(true, map[string]bool{"status": true}, deriveServeRepoPath(t, wt), wt)

	// A sibling of wt is INSIDE repoPath (main) but OUTSIDE startPath (wt).
	// Anchored on repoPath (correct) this resolves; anchored on startPath
	// it would be rejected — so this call distinguishes the two anchors.
	sibling := filepath.Join(main, "pkga")
	result := callTool(t, s, "codegraph_status", map[string]any{"path": sibling})
	if result.IsError {
		t.Fatalf("a path inside repoPath (%q) was rejected — confinement is anchored on startPath, not repoPath: %+v", sibling, result)
	}

	// A genuinely outside, separately-indexed path must still be rejected
	// — confining on repoPath must not be confused with "accept anything".
	outside := copyFixture(t)
	indexFixture(t, outside)
	result = callTool(t, s, "codegraph_status", map[string]any{"path": outside})
	if !result.IsError {
		t.Fatal("a path outside repoPath was accepted — confinement is not enforced")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestIndexAppearingMidSessionRegistersTools is SPEC-05's core promise,
// proven at the Go level: a server built with hasIndex=false against a
// directory that has no .codegraph/ yet advertises exactly zero tools;
// after that same directory gains a real index (indexFixture, a real
// Pebble-backed store via indexer.Run, never a mock), the very next
// tools/list on this SAME *mcp.Server advertises exactly the set a
// server built with hasIndex=true from the start would have — no
// restart, no reconnect. Both assertions use exact set equality
// (equalStrings), never a non-empty or count check.
func TestIndexAppearingMidSessionRegistersTools(t *testing.T) {
	dir := copyFixture(t) // deliberately NOT indexed yet

	companions, _ := ResolveCompanions("", false) // the default surface
	s := BuildServer(false, companions, dir, dir)

	got := listToolNames(t, s)
	if !equalStrings(got, nil) {
		t.Fatalf("registered tools before the index appears = %v, want none", got)
	}

	indexFixture(t, dir)

	got = listToolNames(t, s)
	want := allEightToolNames(t)
	if !equalStrings(got, want) {
		t.Fatalf("registered tools after the index appears = %v, want %v", got, want)
	}
}

// TestIndexAppearingMidSessionHonorsToolFilter proves the re-check
// registers tools through the SAME narrowing filter construction-time
// registration uses, rather than registering everything once an index is
// merely present — the exact set the filter selects, in exact-set-
// equality form, is the only acceptable outcome. Under the default-all
// contract this test carries more weight than it did as
// ...HonorsAllowlist: "register everything on transition" is now the
// DEFAULT behavior, so a re-check that ignored the filter would look
// correct in every other test in this file.
func TestIndexAppearingMidSessionHonorsToolFilter(t *testing.T) {
	dir := copyFixture(t) // deliberately NOT indexed yet

	s := BuildServer(false, map[string]bool{"node": true, "status": true}, dir, dir)

	got := listToolNames(t, s)
	if !equalStrings(got, nil) {
		t.Fatalf("registered tools before the index appears = %v, want none", got)
	}

	indexFixture(t, dir)

	got = listToolNames(t, s)
	want := []string{"codegraph_explore", "codegraph_node", "codegraph_status"}
	if !equalStrings(got, want) {
		t.Fatalf("registered tools after the index appears = %v, want %v", got, want)
	}
}

// TestIndexDisappearingMidSessionUnregistersTools is SPEC-05's reverse
// transition: an indexed server that loses its .codegraph/ directory
// mid-session advertises exactly zero tools on the very next tools/list.
func TestIndexDisappearingMidSessionUnregistersTools(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false) // the default surface
	s := BuildServer(true, companions, dir, dir)

	got := listToolNames(t, s)
	want := allEightToolNames(t)
	if !equalStrings(got, want) {
		t.Fatalf("registered tools = %v, want %v", got, want)
	}

	if err := os.RemoveAll(filepath.Join(dir, ".codegraph")); err != nil {
		t.Fatalf("remove .codegraph: %v", err)
	}

	got = listToolNames(t, s)
	if !equalStrings(got, nil) {
		t.Fatalf("registered tools after the index disappears = %v, want none", got)
	}
}

// TestRepeatedListsDoNotDuplicateTools pins the re-check's flip-guard: a
// re-check that unconditionally called registerTools on every request
// (rather than only on a false-to-true state flip) would still return
// exactly one entry per name (mcp.AddTool replaces a same-named tool
// rather than appending a duplicate), so this test's job is to prove the
// flip-guard exists at all — repeated calls make no additional
// registerTools call once the state is steady, verified indirectly by
// asserting the exact same set on every one of three separate
// listToolNames calls against the same steady-state server.
func TestRepeatedListsDoNotDuplicateTools(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	companions, _ := ResolveCompanions("", false) // the default surface
	s := BuildServer(true, companions, dir, dir)

	want := allEightToolNames(t)
	for i := 0; i < 3; i++ {
		got := listToolNames(t, s)
		if !equalStrings(got, want) {
			t.Fatalf("call %d: registered tools = %v, want %v", i, got, want)
		}
	}
}

// TestSessionLineReflectsPostAppearanceToolCount is VRFY-03/T-03-17's
// mid-session assertion: a server built with hasIndex=false writes
// tools=0 on its first classic initialize (sendRawInitialize, reused from
// session_line_concurrency_test.go — see its doc comment for why the
// classic handshake, not newTestSession's Connect, is required here). The
// index then appears on disk, and a SECOND classic initialize (a fresh
// session, since go-sdk rejects a second "initialize" on the same
// session) writes tools=8 — the re-check ran before this initialize's own
// next() call, so the count reflects a live reading, not the
// construction-time value the first line already proved was 0.
func TestSessionLineReflectsPostAppearanceToolCount(t *testing.T) {
	dir := copyFixture(t) // deliberately NOT indexed yet

	companions, _ := ResolveCompanions("", false) // the default surface

	var log bytes.Buffer
	s := BuildServer(false, companions, dir, dir, WithSessionLog(&log))

	sendRawInitialize(t, s, "codegraph-mcp-test", "0.0.0")

	indexFixture(t, dir)

	sendRawInitialize(t, s, "codegraph-mcp-test", "0.0.0")

	lines := strings.Split(strings.TrimSuffix(log.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d session lines, want 2: %q", len(lines), log.String())
	}

	first, err := parseSessionLineFields(lines[0] + "\n")
	if err != nil {
		t.Fatalf("parseSessionLineFields(first line): %v", err)
	}
	if first["tools"] != "0" {
		t.Fatalf("first session line tools = %q, want %q (pre-appearance)", first["tools"], "0")
	}

	second, err := parseSessionLineFields(lines[1] + "\n")
	if err != nil {
		t.Fatalf("parseSessionLineFields(second line): %v", err)
	}
	wantTools := strconv.Itoa(len(allToolNames()))
	if second["tools"] != wantTools {
		t.Fatalf("second session line tools = %q, want %q (post-appearance, not the construction-time 0)", second["tools"], wantTools)
	}
}
