package mcp

// markdown_test.go closes SURF-06's blind spot (CONTEXT.md D-16/D-17,
// RESEARCH.md's "★ THE BLIND SPOT" section, plan 02-06 Task 1). Before this
// file, ZERO tests asserted the MCP success payload of ANY of
// callers/callees/impact/search/files, and codegraph_status's ONLY MCP
// coverage was its error path (TestOpenEnginePathConfinedToRepoRoot in
// server_test.go). SURF-06 could have been skipped wholesale — or
// implemented wrong in either direction — and `go test ./...` would have
// stayed fully green. That is a literal recurrence of Phase 1's CR-02:
// "implemented + unit-tested + marked complete" is not the same thing as
// "reachable through a real entry point."
//
// Every SURF-06 table row below asserts BOTH:
//   - json.Unmarshal([]byte(text), &v) FAILS — makes a silent regression
//     back to raw json.Marshal impossible;
//   - text contains the tool's expected markdown marker — makes an empty
//     or garbage render impossible.
// Neither assertion alone is sufficient: the negative alone is satisfied by
// an empty/garbage render; the positive alone is satisfied by "marker line,
// then a JSON blob" (i.e. valid markdown text that happens to be followed
// by json.Marshal output). Both together are what the blind spot demands.
//
// ★ Do NOT extend TestExploreCLIMatchesMCP / TestNodeCLIMatchesMCP
// (testdata/golden/golden_parity_test.go) to the 5 SURF-06 tools. That
// harness asserts CLI output == MCP output byte-for-byte, which SURF-06
// makes INTENTIONALLY FALSE for callers/callees/impact/search/files: their
// CLI --json path stays JSON (the golden-parity shape oracle), while MCP
// becomes markdown. The CLI==MCP byte-identity invariant holds only for
// explore/node, whose CLI surface has no --json mode at all
// (internal/cli/{explore,node}.go: "Emits markdown text only — no --json,
// per D-01b"). A future contributor "restoring parity" by making these 5
// byte-identical to the CLI again would be reintroducing the
// JSON-stuffed-into-an-MCP-text-block anti-pattern SURF-06 exists to
// remove.

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// worktreeNoticeText is the fixed, glyph-led prefix of the D-11 compact
// worktree notice (query.WorktreeNotice) — the text prepended to all 7
// non-status read tools on a borrowed-index mismatch.
const worktreeNoticeText = "⚠ CodeGraph results below come from a different git worktree"

// worktreeBlockquotePrefix is the fixed, blockquoted lead line of the D-11
// verbose worktree warning (query.WorktreeWarningBlockquote) — the form
// codegraph_status carries instead of the compact notice.
const worktreeBlockquotePrefix = "> ⚠ This CodeGraph index belongs to a different git working tree."

// callTool runs the MCP initialize handshake against s (in-process, via
// mcpclient.NewInProcessClient, per server_test.go's existing pattern) and
// CallTools name with args, returning the raw result so callers can decide
// how to handle an error result themselves.
func callTool(t *testing.T, s *server.MCPServer, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	c, err := mcpclient.NewInProcessClient(s)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	initClient(t, ctx, c)

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return result
}

// resultText extracts the first text content block from result, failing
// the test if result has no content or the first block is not text.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("CallTool result has no content")
	}
	text, ok := mcp.AsTextContent(result.Content[0])
	if !ok {
		t.Fatalf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text
}

// runGitM mirrors internal/query/engine_worktree_test.go's runGitW. Test
// files are not importable across packages, so this is a minimal local
// copy of the same hermetic-flags-plus-skip-on-failure shape (D-15): any
// git failure, including git being absent from PATH, skips the calling
// test (t.Skip, never t.Fatal) — WORK-03's best-effort philosophy applies
// to the fixtures that exercise it too.
func runGitM(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}

// deriveServeRepoPath replicates internal/cli/serve.go's RunE verbatim —
// the EXACT repoPath derivation CR-01 fixes: repoPath starts as start and
// is overwritten with the resolved index root only when
// query.ResolveCodegraphDir finds one at or above start. This is the
// literal computation BuildServer's only production caller performs.
// Before CR-01, every test in this file that exercised the mismatch
// fixture instead hand-picked the worktree itself as BuildServer's
// repoPath — a value serve.go can never produce, since it always
// overwrites `start` with ResolveCodegraphDir's output once an index is
// found. A test that calls BuildServer with a literal it chose itself
// cannot prove this feature is reachable through the real entry point;
// routing every fixture-driven BuildServer call through this helper
// closes that gap (CR-01's "required test").
func deriveServeRepoPath(t *testing.T, start string) string {
	t.Helper()
	repoPath := start
	if dir, err := query.ResolveCodegraphDir(start); err == nil {
		repoPath = dir
	}
	return repoPath
}

// mcpWorktreeMismatchFixture builds a real, indexed main checkout plus a
// linked worktree nested at .claude/worktrees/probe — D-15's motivating
// true-positive layout, mirroring internal/query/engine_worktree_test.go's
// worktreeMismatchFixture (a second, package-local copy since Go test
// helpers are not importable across packages). Returns both absolute
// paths.
//
// ★ Confinement interaction (CR-01/CR-02): every test in this file that
// uses this fixture now constructs its server via
// BuildServer(true, allowlist, deriveServeRepoPath(t, wt), wt) — repoPath
// (the confinement root) is the RESOLVED index root (main, found by
// walking up from wt, since wt has no .codegraph/ of its own), and
// startPath (the handlers' default path) is wt itself, exactly mirroring
// serve.go's real derivation. Tool calls issue with no explicit "path"
// argument, so resolvePath defaults to startPath (wt) and
// confineToRepoRoot then checks wt against repoPath (main) — this always
// succeeds structurally, because ResolveCodegraphDir's upward walk
// guarantees repoPath is wt itself or an ancestor of it (see
// mcp.BuildServer's doc comment). No confinement weakening was required.
func mcpWorktreeMismatchFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()

	main := copyFixture(t)
	runGitM(t, main, "init")
	runGitM(t, main, "add", "-A")
	runGitM(t, main, "commit", "-m", "init")

	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitM(t, main, "worktree", "add", "-b", "probe", wt)

	indexFixture(t, main)

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

// TestMarkdownOutput is Test 1 (SURF-06, the blind-spot closer): each of
// the 5 JSON-shaped MCP read tools, driven through the real CallTool path
// against an indexed fixture, returns text that both fails json.Unmarshal
// and contains its expected markdown marker.
func TestMarkdownOutput(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	allowlist := map[string]bool{"search": true, "callers": true, "callees": true, "impact": true, "files": true}
	s := BuildServer(true, allowlist, dir, dir)

	locationTableMarker := "| Name | Kind | Location |"

	cases := []struct {
		name   string
		tool   string
		args   map[string]any
		marker string
	}{
		// helper is called by pkga.Alpha (gofixture/pkga/pkga.go) — a
		// non-empty callers result, so the assertion exercises a real
		// table, not the empty-set sentence.
		{"callers", "codegraph_callers", map[string]any{"symbol": "helper"}, locationTableMarker},
		// Alpha calls helper — a non-empty callees result.
		{"callees", "codegraph_callees", map[string]any{"symbol": "Alpha"}, locationTableMarker},
		// helper's reverse blast radius includes Alpha — non-empty.
		{"impact", "codegraph_impact", map[string]any{"symbol": "helper"}, locationTableMarker},
		{"search", "codegraph_search", map[string]any{"query": "Alpha"}, locationTableMarker},
		{"files", "codegraph_files", map[string]any{}, "| Path | Language | Nodes | Edges |"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := callTool(t, s, tc.tool, tc.args)
			if result.IsError {
				t.Fatalf("%s returned an error result: %+v", tc.tool, result)
			}
			text := resultText(t, result)

			var v any
			if err := json.Unmarshal([]byte(text), &v); err == nil {
				t.Fatalf("%s output is valid JSON (want markdown, SURF-06 not yet delivered): %q", tc.tool, text)
			}
			if !strings.Contains(text, tc.marker) {
				t.Fatalf("%s output missing expected markdown marker %q, got: %q", tc.tool, tc.marker, text)
			}
		})
	}
}

// TestStatusMarkdownOutput is Test 2 (D-17): codegraph_status returns text
// that fails json.Unmarshal and contains its MCP-form bolded header,
// proving status moved off MarshalStatusJSON onto its own
// RenderStatusMarkdown, not onto one of the 5 Location-shaped renderers.
func TestStatusMarkdownOutput(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	s := BuildServer(true, map[string]bool{"status": true}, dir, dir)

	result := callTool(t, s, "codegraph_status", map[string]any{})
	if result.IsError {
		t.Fatalf("codegraph_status returned an error result: %+v", result)
	}
	text := resultText(t, result)

	var v any
	if err := json.Unmarshal([]byte(text), &v); err == nil {
		t.Fatalf("codegraph_status output is valid JSON (want markdown, D-17 not yet delivered): %q", text)
	}
	if !strings.Contains(text, "**CodeGraph Status**") {
		t.Fatalf("codegraph_status output missing expected marker %q, got: %q", "**CodeGraph Status**", text)
	}
}

// TestNoWorktreeNoticeOnCleanTree is Test 3 (WORK-02/D-12, clean case): on
// an ordinary indexed fixture with no borrowed worktree, none of the 8 MCP
// read tools' output contains either worktree-mismatch string — a clean
// tree must add nothing to any tool's output.
func TestNoWorktreeNoticeOnCleanTree(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)

	allowlist := map[string]bool{"node": true, "search": true, "callers": true, "callees": true, "impact": true, "files": true, "status": true}
	s := BuildServer(true, allowlist, dir, dir)

	cases := []struct {
		tool string
		args map[string]any
	}{
		{"codegraph_explore", map[string]any{"query": "main"}},
		{"codegraph_node", map[string]any{"symbol": "Alpha"}},
		{"codegraph_search", map[string]any{"query": "Alpha"}},
		{"codegraph_callers", map[string]any{"symbol": "helper"}},
		{"codegraph_callees", map[string]any{"symbol": "Alpha"}},
		{"codegraph_impact", map[string]any{"symbol": "helper"}},
		{"codegraph_files", map[string]any{}},
		{"codegraph_status", map[string]any{}},
	}

	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			result := callTool(t, s, tc.tool, tc.args)
			if result.IsError {
				t.Fatalf("%s returned an error result: %+v", tc.tool, result)
			}
			text := resultText(t, result)
			if strings.Contains(text, "different git worktree") || strings.Contains(text, "different git working tree") {
				t.Fatalf("%s output contains a worktree-mismatch string on a clean (non-worktree) tree, want none: %q", tc.tool, text)
			}
		})
	}
}

// TestWorktreeNoticeOnMismatch is Test 4 (WORK-02/D-12, mismatch case):
// against a real .claude/worktrees/ fixture whose index resolves to the
// main checkout, each of the 7 non-status tools' output STARTS WITH the
// compact notice, and codegraph_status's output carries the blockquoted
// verbose warning instead — never the compact form.
func TestWorktreeNoticeOnMismatch(t *testing.T) {
	wt, _ := mcpWorktreeMismatchFixture(t)

	allowlist := map[string]bool{"node": true, "search": true, "callers": true, "callees": true, "impact": true, "files": true, "status": true}
	s := BuildServer(true, allowlist, deriveServeRepoPath(t, wt), wt)

	nonStatus := []struct {
		tool string
		args map[string]any
	}{
		{"codegraph_explore", map[string]any{"query": "main"}},
		{"codegraph_node", map[string]any{"symbol": "Alpha"}},
		{"codegraph_search", map[string]any{"query": "Alpha"}},
		{"codegraph_callers", map[string]any{"symbol": "helper"}},
		{"codegraph_callees", map[string]any{"symbol": "Alpha"}},
		{"codegraph_impact", map[string]any{"symbol": "helper"}},
		{"codegraph_files", map[string]any{}},
	}

	for _, tc := range nonStatus {
		t.Run(tc.tool, func(t *testing.T) {
			result := callTool(t, s, tc.tool, tc.args)
			if result.IsError {
				t.Fatalf("%s returned an error result: %+v", tc.tool, result)
			}
			text := resultText(t, result)
			if !strings.HasPrefix(text, worktreeNoticeText) {
				t.Fatalf("%s output does not start with the compact worktree notice %q, got: %q", tc.tool, worktreeNoticeText, text)
			}
		})
	}

	t.Run("codegraph_status", func(t *testing.T) {
		result := callTool(t, s, "codegraph_status", map[string]any{})
		if result.IsError {
			t.Fatalf("codegraph_status returned an error result: %+v", result)
		}
		text := resultText(t, result)
		if strings.HasPrefix(text, worktreeNoticeText) {
			t.Fatalf("codegraph_status output starts with the COMPACT notice; status must carry the verbose blockquote instead, got: %q", text)
		}
		if !strings.Contains(text, worktreeBlockquotePrefix) {
			t.Fatalf("codegraph_status output missing the blockquoted verbose worktree warning %q, got: %q", worktreeBlockquotePrefix, text)
		}
	})
}

// TestWorktreeNoticeNotAppliedOnError is Test 5 (D-12, error results): a
// tool call that produces an error result (a required-arg violation, here
// codegraph_callers with no "symbol") is NOT prefixed with the notice —
// the notice requires a live Engine, which a pre-openEngine arg-validation
// failure never reaches.
func TestWorktreeNoticeNotAppliedOnError(t *testing.T) {
	wt, _ := mcpWorktreeMismatchFixture(t)

	s := BuildServer(true, map[string]bool{"callers": true}, deriveServeRepoPath(t, wt), wt)

	result := callTool(t, s, "codegraph_callers", map[string]any{})
	if !result.IsError {
		t.Fatal("codegraph_callers with no symbol arg: expected an error result, got success")
	}
	text := resultText(t, result)
	if strings.Contains(text, worktreeNoticeText) || strings.HasPrefix(text, "⚠") {
		t.Fatalf("error result was prefixed with the worktree notice, want none: %q", text)
	}
}

// TestWorktreeNoticeConsistentAcrossCalls is Test 6 (D-13): two successive
// CallTools against the SAME server share one detection — asserted as
// observable behavior (the notice text is identical across both calls),
// not by reaching into the detector or counting subprocesses. The
// server-scoped cache's per-server construction is verified structurally
// by Task 3's acceptance criteria; this test only pins the user-visible
// contract that D-13 exists to guarantee: a long-lived server's notice
// does not flap between calls.
func TestWorktreeNoticeConsistentAcrossCalls(t *testing.T) {
	wt, _ := mcpWorktreeMismatchFixture(t)

	s := BuildServer(true, map[string]bool{}, deriveServeRepoPath(t, wt), wt)

	result1 := callTool(t, s, "codegraph_explore", map[string]any{"query": "main"})
	result2 := callTool(t, s, "codegraph_explore", map[string]any{"query": "main"})
	if result1.IsError || result2.IsError {
		t.Fatalf("codegraph_explore returned an error result: %+v / %+v", result1, result2)
	}
	text1 := resultText(t, result1)
	text2 := resultText(t, result2)

	idx1 := strings.Index(text1, "\n\n")
	idx2 := strings.Index(text2, "\n\n")
	if idx1 < 0 || idx2 < 0 {
		t.Fatalf("expected a notice-then-blank-line prefix in both outputs, got: %q / %q", text1, text2)
	}
	notice1, notice2 := text1[:idx1], text2[:idx2]

	if !strings.HasPrefix(notice1, worktreeNoticeText) {
		t.Fatalf("expected the worktree notice to be present, got: %q", notice1)
	}
	if notice1 != notice2 {
		t.Fatalf("two calls against the same server produced different worktree notices: %q vs %q — D-13's server-scoped cache should make detection deterministic within one server's lifetime", notice1, notice2)
	}
}

// TestCancelledCallDoesNotPoisonNoticeForSubsequentCalls is BL-01's
// required test (02-REVIEW-2.md): WR-01 threaded the caller's real,
// cancelable ctx all the way down to gitmeta.DetectIndexMismatch's git
// subprocesses; WR-02 made the detector server-scoped and long-lived.
// Together, a SINGLE cancelled tool call collapses the aborted git spawn
// into gate 1's "" -> nil — a verdict indistinguishable from "checked, no
// mismatch" — and (pre-fix) CachingDetector cached that nil FOREVER,
// silently disabling the worktree notice for the rest of the server's
// life. This reproduces the reviewer's proof through the REAL
// BuildServer -> CallTool handler path (not a unit test on the cache
// alone, per 02-REVIEW-2.md's acceptance bar): call 1 issues with an
// ALREADY-CANCELLED context; call 2 is an ordinary, healthy call against
// the SAME server and must still see the notice.
func TestCancelledCallDoesNotPoisonNoticeForSubsequentCalls(t *testing.T) {
	wt, _ := mcpWorktreeMismatchFixture(t)

	s := BuildServer(true, map[string]bool{}, deriveServeRepoPath(t, wt), wt)

	// Sanity: a fresh server DOES emit the notice, so a later failure to
	// see it is attributable to call 1's cancellation, not a broken
	// fixture.
	sanity := callTool(t, s, "codegraph_explore", map[string]any{"query": "main"})
	if sanity.IsError || !strings.HasPrefix(resultText(t, sanity), worktreeNoticeText) {
		t.Fatalf("sanity check: fresh server does not emit the worktree notice: %+v", sanity)
	}

	// A SECOND, independent server (same fixture) isolates call 1's
	// cancellation from the sanity check above — each gets its own
	// detector/cache.
	s2 := BuildServer(true, map[string]bool{}, deriveServeRepoPath(t, wt), wt)

	c, err := mcpclient.NewInProcessClient(s2)
	if err != nil {
		t.Fatalf("NewInProcessClient: %v", err)
	}
	defer c.Close()

	// The MCP initialize handshake must succeed on a live context —
	// only the TOOL call below carries the cancelled one.
	initClient(t, context.Background(), c)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Call 1: an ALREADY-CANCELLED context. openEngine still succeeds
	// (query.OpenAt takes no ctx), but eng.WorktreeMismatch(ctx)'s git
	// subprocess is aborted instantly by exec.CommandContext, so
	// DetectIndexMismatch degrades to nil for THIS call (WORK-03: never
	// block or error a read on a failed git probe) — that degradation
	// itself is correct and expected, and is NOT what this test pins.
	if _, err := c.CallTool(cancelledCtx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "codegraph_explore",
			Arguments: map[string]any{"query": "main"},
		},
	}); err != nil {
		t.Fatalf("CallTool (cancelled ctx): %v", err)
	}

	// Call 2: a normal, healthy call against the SAME server (s2) — same
	// detector, same cache. Pre-fix, this silently lost the notice
	// forever because call 1's cancelled-ctx verdict was cached as a
	// permanent false "no mismatch".
	result2 := callTool(t, s2, "codegraph_explore", map[string]any{"query": "main"})
	if result2.IsError {
		t.Fatalf("codegraph_explore (call 2, healthy) returned an error result: %+v", result2)
	}
	text2 := resultText(t, result2)
	if !strings.HasPrefix(text2, worktreeNoticeText) {
		t.Fatalf("BL-01 REGRESSION: after ONE cancelled tool call, this server no longer emits the worktree notice on a subsequent healthy call. got: %q", text2)
	}
}
