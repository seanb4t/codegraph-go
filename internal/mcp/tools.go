package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// resolvePath returns req's "path" arg, falling back to defaultPath when
// the caller omits it — mirroring the CLI's -p/--path default-cwd
// behavior (D-01a), but here defaulting to the server's configured repo
// root instead of a live process cwd.
func resolvePath(req mcp.CallToolRequest, defaultPath string) string {
	return req.GetString("path", defaultPath)
}

// confineToRepoRoot resolves path against the server's configured
// repoPath and rejects it (CR-02, trust-boundary defense) unless it
// resolves to repoPath itself or a descendant of it. An MCP client —
// which in this product's threat model may be an AI agent processing
// attacker-influenced content — must not be able to redirect a tool call
// to an entirely different .codegraph/-indexed project elsewhere on the
// host filesystem merely by supplying a "path" argument.
func confineToRepoRoot(path, repoPath string) (string, error) {
	repoAbs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(repoAbs, pathAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("mcp: path %q is outside the server's configured repo root", path)
	}
	return pathAbs, nil
}

// openEngine is every handler's single read seam: it resolves req's
// "path" arg against defaultPath, confines it to the server's repo root
// (CR-02, confineToRepoRoot — this check runs BEFORE query.OpenAt and
// must never move or weaken, since a caller-supplied "path" could
// otherwise redirect detection, not just reads, outside the server's
// configured root), opens a FRESH query.OpenAt snapshot for this call
// (D-02/D-08b, RESEARCH Pitfall 2 — never a snapshot cached at server
// construction), and installs the server-scoped detector (D-13) so this
// call's worktree detection shares the one cache BuildServer constructed
// rather than probing git uncached. The caller owns closing the returned
// io.Closer.
func openEngine(req mcp.CallToolRequest, defaultPath string, detector *gitmeta.CachingDetector) (*query.Engine, func() error, error) {
	path := resolvePath(req, defaultPath)
	confined, err := confineToRepoRoot(path, defaultPath)
	if err != nil {
		return nil, nil, err
	}
	eng, closer, err := query.OpenAt(confined)
	if err != nil {
		return nil, nil, err
	}
	eng.UseDetector(detector)
	return eng, closer.Close, nil
}

// exploreTool describes codegraph_explore's schema, mirroring the CLI's
// explore flags (query positional, -p/--path, --max-files) — D-08b.
func exploreTool() mcp.Tool {
	return mcp.NewTool("codegraph_explore",
		mcp.WithDescription("Explore relevant symbols: verbatim source, call paths, blast radius"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural-language or symbol/file query")),
		mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		mcp.WithNumber("max_files", mcp.Description("Cap on distinct files returned (default 5)")),
	)
}

// exploreHandler resolves its args, opens a fresh engine snapshot, and
// delegates to Engine.Explore — the exact CLI explore code path — then
// returns the markdown result, compact-worktree-notice-prefixed on the
// success path (WORK-02/D-12; no-op on the mismatch-free case since
// query.WorktreeNotice returns "" — no re-rendering in internal/mcp,
// D-08b).
func exploreHandler(defaultPath string, detector *gitmeta.CachingDetector) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxFiles := req.GetInt("max_files", 0)

		eng, close, err := openEngine(req, defaultPath, detector)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer close()

		out, err := eng.Explore(q, maxFiles)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
	}
}

// companionTool returns the mcp.Tool schema for one of the 7
// CODEGRAPH_MCP_TOOLS-gated companion tools, with argument schemas
// mirroring the corresponding CLI command's flags (D-08b). Panics on an
// unrecognized name — callers only ever pass names from companionNames.
func companionTool(name string) mcp.Tool {
	switch name {
	case "node":
		return mcp.NewTool("codegraph_node",
			mcp.WithDescription("Show a symbol's signature, calls, and callers, or a line-numbered file read"),
			mcp.WithString("symbol", mcp.Description("Symbol name to look up (omit for a file-mode read)")),
			mcp.WithString("file", mcp.Description("File path — disambiguates symbol, or selects file-mode when symbol is omitted")),
			mcp.WithNumber("line", mcp.Description("Line number — narrows an overloaded symbol to the definition containing (or nearest) this line (NODE-03)")),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "search":
		return mcp.NewTool("codegraph_search",
			mcp.WithDescription("Lexically search symbol names/qualified names, returning locations only"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search term")),
			mcp.WithString("kind", mcp.Description("Restrict to one node kind")),
			mcp.WithNumber("limit", mcp.Description("Cap on results returned")),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "callers":
		return mcp.NewTool("codegraph_callers",
			mcp.WithDescription("List a symbol's reverse callers"),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol name")),
			mcp.WithNumber("limit", mcp.Description("Cap on results returned")),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "callees":
		return mcp.NewTool("codegraph_callees",
			mcp.WithDescription("List a symbol's forward call targets"),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol name")),
			mcp.WithNumber("limit", mcp.Description("Cap on results returned")),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "impact":
		return mcp.NewTool("codegraph_impact",
			mcp.WithDescription("Depth-bounded reverse blast radius of a symbol"),
			mcp.WithString("symbol", mcp.Required(), mcp.Description("Symbol name")),
			mcp.WithNumber("depth", mcp.Description("BFS depth (default 5, max 50)")),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "files":
		return mcp.NewTool("codegraph_files",
			mcp.WithDescription("Browse the indexed file structure"),
			mcp.WithString("pattern", mcp.Description("Shell glob narrowing the result set")),
			mcp.WithString("filter", mcp.Description("Restrict to one language")),
			mcp.WithNumber("depth", mcp.Description("Directory-nesting cap (0 = unlimited)")),
			mcp.WithString("format", mcp.Description(`"flat" (default) or "tree"`)),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	case "status":
		return mcp.NewTool("codegraph_status",
			mcp.WithDescription("Report index health and counts"),
			mcp.WithString("path", mcp.Description("Repo path (default: server cwd)")),
		)
	default:
		panic("mcp: companionTool: unknown tool name " + name)
	}
}

// companionHandler returns the ToolHandlerFunc for one of the 7
// companion tools. Every branch follows the same shape: parse args,
// open a fresh engine snapshot (openEngine), delegate to the matching
// Engine method, render via the matching internal/query Render*Markdown
// formatter (D-08b — no rendering re-implemented here), and return the
// markdown text. Panics on an unrecognized name — callers only ever
// pass names from companionNames.
//
// SURF-06 (D-16) moved this file's six call sites (search/callers/
// callees/impact/files/status) from json.Marshal / Marshal*JSON to the
// matching Render*Markdown function. This creates a deliberate,
// intentional asymmetry that must NOT be "unified": after this change,
// each Marshal*JSON helper (traverse.go, files.go, status.go) has
// exactly one caller — the CLI --json path, whose consumer is a parser
// (jq, scripts, CI) and which is also testdata/golden's shape oracle —
// while each Render*Markdown function has exactly one caller — this
// file, whose consumer is a language model, not a parser. This closes a
// Go-vs-TS divergence: TS returns markdown from every MCP tool; our
// JSON-shaped tools were the anomaly.
//
// Six of the seven branches (every one except "status") additionally
// prefix the compact worktree notice (query.WorktreeNotice,
// WORK-02/D-12) onto the SUCCESS return only — every branch's failure
// paths return through mcp.NewToolResultError BEFORE reaching that
// point, so "no-op on isError results" holds structurally, with no
// redundant isError check needed. "status" is deliberately excluded:
// query.RenderStatusMarkdown already embeds its own verbose blockquote
// warning (D-17), and a second compact prefix would duplicate it.
func companionHandler(name, defaultPath string, detector *gitmeta.CachingDetector) server.ToolHandlerFunc {
	switch name {
	case "node":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol := req.GetString("symbol", "")
			file := req.GetString("file", "")
			// NODE-03: 0 (GetInt's zero-value default) means "no line hint" —
			// not a valid 1-indexed source line — matching Engine.Node's own
			// nil-means-unset convention (EXPL-05/NODE-04: keeps this handler
			// a thin delegation to the same Engine the CLI uses).
			line := req.GetInt("line", 0)
			var lineHint *int
			if line != 0 {
				lineHint = &line
			}

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			out, err := eng.Node(symbol, file, lineHint)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "search":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			term, err := req.RequireString("query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			kind := req.GetString("kind", "")
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			locs, err := eng.Search(term, kind, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderSearchMarkdown(term, locs)
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "callers":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Callers(symbol, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderCallersMarkdown(result)
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "callees":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Callees(symbol, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderCalleesMarkdown(result)
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "impact":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			depth := req.GetInt("depth", 0)

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Impact(symbol, depth)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderImpactMarkdown(result)
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "files":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			opts := query.FilesOptions{
				Pattern: req.GetString("pattern", ""),
				Filter:  req.GetString("filter", ""),
				Depth:   req.GetInt("depth", 0),
				Format:  req.GetString("format", ""),
			}

			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Files(opts)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderFilesMarkdown(result)
			return mcp.NewToolResultText(query.WorktreeNotice(eng.WorktreeMismatch()) + out), nil
		}
	case "status":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			eng, close, err := openEngine(req, defaultPath, detector)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			// codegraph_status is EXCLUDED from the compact notice
			// (WORK-02/D-12): Engine.Status() already computes
			// StatusResult.WorktreeMismatch, and RenderStatusMarkdown
			// embeds it as its own verbose blockquote — mirroring TS's
			// withWorktreeNotice, which excludes codegraph_status for
			// exactly this reason (it carries its own verbose form
			// instead of the compact one).
			result, err := eng.Status()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out := query.RenderStatusMarkdown(result)
			return mcp.NewToolResultText(out), nil
		}
	default:
		panic("mcp: companionHandler: unknown tool name " + name)
	}
}
