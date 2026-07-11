package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// resolvePath returns req's "path" arg, falling back to defaultPath when
// the caller omits it — mirroring the CLI's -p/--path default-cwd
// behavior (D-01a), but here defaulting to the server's configured repo
// root instead of a live process cwd.
func resolvePath(req mcp.CallToolRequest, defaultPath string) string {
	return req.GetString("path", defaultPath)
}

// openEngine is every handler's single read seam: it resolves req's
// "path" arg against defaultPath and opens a FRESH query.OpenAt
// snapshot for this call (D-02/D-08b, RESEARCH Pitfall 2 — never a
// snapshot cached at server construction). The caller owns closing the
// returned io.Closer.
func openEngine(req mcp.CallToolRequest, defaultPath string) (*query.Engine, func() error, error) {
	path := resolvePath(req, defaultPath)
	eng, closer, err := query.OpenAt(path)
	if err != nil {
		return nil, nil, err
	}
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
// returns the markdown result as-is (no re-rendering in internal/mcp,
// D-08b).
func exploreHandler(defaultPath string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		q, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		maxFiles := req.GetInt("max_files", 0)

		eng, close, err := openEngine(req, defaultPath)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		defer close()

		out, err := eng.Explore(q, maxFiles)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(out), nil
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
// Engine method, marshal via the matching internal/query Marshal*JSON
// formatter (D-08b — no JSON shaping re-implemented here), and return
// the JSON text. Panics on an unrecognized name — callers only ever
// pass names from companionNames.
func companionHandler(name, defaultPath string) server.ToolHandlerFunc {
	switch name {
	case "node":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol := req.GetString("symbol", "")
			file := req.GetString("file", "")

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			out, err := eng.Node(symbol, file)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(out), nil
		}
	case "search":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			term, err := req.RequireString("query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			kind := req.GetString("kind", "")
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			locs, err := eng.Search(term, kind, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			// Search's []query.Location result carries its own json tags
			// (search.go) — no dedicated MarshalSearchJSON exists (unlike
			// its sibling commands), so this is a thin encoding/json.Marshal
			// pass over an already-fully-shaped internal/query type, not a
			// second rendering path (D-08b).
			data, err := json.Marshal(locs)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	case "callers":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Callers(symbol, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, err := query.MarshalCallersJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	case "callees":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			limit := req.GetInt("limit", 0)

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Callees(symbol, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, err := query.MarshalCalleesJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	case "impact":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			symbol, err := req.RequireString("symbol")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			depth := req.GetInt("depth", 0)

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Impact(symbol, depth)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, err := query.MarshalImpactJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	case "files":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			opts := query.FilesOptions{
				Pattern: req.GetString("pattern", ""),
				Filter:  req.GetString("filter", ""),
				Depth:   req.GetInt("depth", 0),
				Format:  req.GetString("format", ""),
			}

			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Files(opts)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, err := query.MarshalFilesJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	case "status":
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			eng, close, err := openEngine(req, defaultPath)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			defer close()

			result, err := eng.Status()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			data, err := query.MarshalStatusJSON(result)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}
	default:
		panic("mcp: companionHandler: unknown tool name " + name)
	}
}
