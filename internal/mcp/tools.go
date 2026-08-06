package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// resolvePath returns argPath, falling back to defaultPath when the caller
// omits it — mirroring the CLI's -p/--path default-cwd behavior (D-01a),
// but here defaulting to the server's configured repo root instead of a
// live process cwd.
func resolvePath(argPath, defaultPath string) string {
	if argPath == "" {
		return defaultPath
	}
	return argPath
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

// openEngine is every handler's single read seam: it resolves argPath
// against defaultPath (the CALLER's actual starting directory, CR-01 —
// serve.go's `start`, distinct from repoPath), confines the resolved path
// to repoPath, the server's confinement root (CR-02, confineToRepoRoot —
// this check runs BEFORE query.OpenAt and must never move or weaken, since
// a caller-supplied "path" could otherwise redirect detection, not just
// reads, outside the server's configured root; the DEFAULT defaultPath
// always passes this check because repoPath is
// ResolveCodegraphDir(defaultPath)'s own return value — defaultPath is
// always repoPath itself or a descendant of it, by construction — see
// BuildServer's doc comment), opens a FRESH query.OpenAt snapshot for this
// call (D-02/D-08b, RESEARCH Pitfall 2 — never a snapshot cached at server
// construction), and installs the server-scoped detector (D-13) so this
// call's worktree detection shares the one cache BuildServer constructed
// rather than probing git uncached. The caller owns closing the returned
// io.Closer.
func openEngine(argPath, defaultPath, repoPath string, detector *gitmeta.CachingDetector) (*query.Engine, func() error, error) {
	path := resolvePath(argPath, defaultPath)
	confined, err := confineToRepoRoot(path, repoPath)
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

// toolAnnotations returns the same ToolAnnotations values mark3labs'
// mcp.NewTool defaulted every tool to (readOnlyHint:false,
// destructiveHint:true, idempotentHint:false, openWorldHint:true) —
// carried over byte-for-byte per 02-CONTEXT.md's "Claude's Discretion"
// resolution: these are mark3labs zero-values, not deliberate per-tool
// choices, and arguably wrong for read-only query tools, but correcting
// them is a semantic change out of scope for this phase (Phase 3
// territory, per 02-CONTEXT.md <deferred>).
func toolAnnotations() *mcp.ToolAnnotations {
	destructive := true
	openWorld := true
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: &destructive,
		IdempotentHint:  false,
		OpenWorldHint:   &openWorld,
	}
}

// exploreTool describes codegraph_explore's schema, mirroring the CLI's
// explore flags (query positional, -p/--path, --max-files) — D-08b. The
// input schema itself is left nil: mcp.AddTool infers it from ExploreArgs'
// struct tags (D-07).
func exploreTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "codegraph_explore",
		Description: "Explore relevant symbols: verbatim source, call paths, blast radius",
		Annotations: toolAnnotations(),
	}
}

// ExploreArgs is codegraph_explore's input schema (D-07), inferred by
// mcp.AddTool from these struct tags. Fields are declared in alphabetical
// order of their JSON name (max_files, path, query) so jsonschema-go's
// Go-struct-field-declaration-order property emission matches mark3labs'
// alphabetical order, minimizing plan 02-05's diff review surface. Query
// carries no omitempty — it is the only required field.
type ExploreArgs struct {
	MaxFiles int    `json:"max_files,omitempty" jsonschema:"Cap on distinct files returned (default 5)"`
	Path     string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Query    string `json:"query" jsonschema:"Natural-language or symbol/file query"`
}

// exploreHandler resolves its args, opens a fresh engine snapshot, and
// delegates to Engine.Explore — the exact CLI explore code path — then
// returns the markdown result, compact-worktree-notice-prefixed on the
// success path (WORK-02/D-12; no-op on the mismatch-free case since
// query.WorktreeNotice returns "" — no re-rendering in internal/mcp,
// D-08b). repoPath is the confinement root; defaultPath is the caller's
// start path (CR-01) — see BuildServer's doc comment for why they must
// stay distinct.
//
// WR-04 (02-REVIEW-2.md): parameter order is (repoPath, defaultPath),
// matching BuildServer's own (repoPath, startPath) declared order, so its
// call site below reads the same way its signature does — before this fix
// BuildServer(hasIndex, allowlist, repoPath, startPath) called
// exploreHandler(startPath, repoPath, ...), an inverted order on two
// adjacent same-typed strings that no test distinguished.
//
// SDK-04: a required-argument validation failure (a missing "query") never
// reaches this handler at all — go-sdk's AddTool validates CallToolRequest
// arguments against ExploreArgs' inferred schema before any handler body
// runs (02-RESEARCH.md Q5), so the RequireString-style check this handler
// used to open with is gone; every remaining error path simply returns the
// Go error, which toolForErr converts into the same IsError:true/Content
// shape mcp.NewToolResultError used to build by hand.
func exploreHandler(repoPath, defaultPath string, detector *gitmeta.CachingDetector) func(context.Context, *mcp.CallToolRequest, ExploreArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ExploreArgs) (*mcp.CallToolResult, any, error) {
		eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
		if err != nil {
			return nil, nil, err
		}
		defer close()

		out, err := eng.Explore(args.Query, args.MaxFiles)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
		}, nil, nil
	}
}

// NodeArgs is codegraph_node's input schema (D-07). Fields declared
// alphabetically by JSON name (file, line, path, symbol). Symbol carries
// omitempty — node's file-mode read means symbol is optional, matching
// today's schema (never marked required).
type NodeArgs struct {
	File   string `json:"file,omitempty" jsonschema:"File path — disambiguates symbol, or selects file-mode when symbol is omitted"`
	Line   int    `json:"line,omitempty" jsonschema:"Line number — narrows an overloaded symbol to the definition containing (or nearest) this line (NODE-03)"`
	Path   string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Symbol string `json:"symbol,omitempty" jsonschema:"Symbol name to look up (omit for a file-mode read)"`
}

// SearchArgs is codegraph_search's input schema (D-07). Fields declared
// alphabetically by JSON name (kind, limit, path, query). Query carries no
// omitempty — it is the only required field.
type SearchArgs struct {
	Kind  string `json:"kind,omitempty" jsonschema:"Restrict to one node kind"`
	Limit int    `json:"limit,omitempty" jsonschema:"Cap on results returned"`
	Path  string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Query string `json:"query" jsonschema:"Search term"`
}

// CallersArgs is codegraph_callers' input schema (D-07). Fields declared
// alphabetically by JSON name (limit, path, symbol). Symbol carries no
// omitempty — it is the only required field.
type CallersArgs struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Cap on results returned"`
	Path   string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Symbol string `json:"symbol" jsonschema:"Symbol name"`
}

// CalleesArgs is codegraph_callees' input schema (D-07). Fields declared
// alphabetically by JSON name (limit, path, symbol). Symbol carries no
// omitempty — it is the only required field.
type CalleesArgs struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"Cap on results returned"`
	Path   string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Symbol string `json:"symbol" jsonschema:"Symbol name"`
}

// ImpactArgs is codegraph_impact's input schema (D-07). Fields declared
// alphabetically by JSON name (depth, path, symbol). Symbol carries no
// omitempty — it is the only required field.
type ImpactArgs struct {
	Depth  int    `json:"depth,omitempty" jsonschema:"BFS depth (default 2, max 50)"`
	Path   string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Symbol string `json:"symbol" jsonschema:"Symbol name"`
}

// FilesArgs is codegraph_files' input schema (D-07). Fields declared
// alphabetically by JSON name (depth, filter, format, path, pattern). No
// field is required.
type FilesArgs struct {
	Depth   int    `json:"depth,omitempty" jsonschema:"Directory-nesting cap (0 = unlimited)"`
	Filter  string `json:"filter,omitempty" jsonschema:"Restrict to one language"`
	Format  string `json:"format,omitempty" jsonschema:"\"flat\" (default) or \"tree\""`
	Path    string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
	Pattern string `json:"pattern,omitempty" jsonschema:"Shell glob narrowing the result set"`
}

// StatusArgs is codegraph_status's input schema (D-07). No field is
// required.
type StatusArgs struct {
	Path string `json:"path,omitempty" jsonschema:"Repo path (default: server cwd)"`
}

// companionTool returns the *mcp.Tool schema for one of the 7
// CODEGRAPH_MCP_TOOLS-gated companion tools, with argument schemas
// mirroring the corresponding CLI command's flags (D-08b). Each tool's
// InputSchema is left nil — mcp.AddTool infers it from the matching
// *Args struct's tags (D-07), applied at the registration call site in
// companionHandler. Panics on an unrecognized name — callers only ever
// pass names from companionNames.
func companionTool(name string) *mcp.Tool {
	switch name {
	case "node":
		return &mcp.Tool{
			Name:        "codegraph_node",
			Description: "Show a symbol's signature, calls, and callers, or a line-numbered file read",
			Annotations: toolAnnotations(),
		}
	case "search":
		return &mcp.Tool{
			Name:        "codegraph_search",
			Description: "Lexically search symbol names/qualified names, returning locations only",
			Annotations: toolAnnotations(),
		}
	case "callers":
		return &mcp.Tool{
			Name:        "codegraph_callers",
			Description: "List a symbol's reverse callers",
			Annotations: toolAnnotations(),
		}
	case "callees":
		return &mcp.Tool{
			Name:        "codegraph_callees",
			Description: "List a symbol's forward call targets",
			Annotations: toolAnnotations(),
		}
	case "impact":
		return &mcp.Tool{
			Name:        "codegraph_impact",
			Description: "Depth-bounded reverse blast radius of a symbol",
			Annotations: toolAnnotations(),
		}
	case "files":
		return &mcp.Tool{
			Name:        "codegraph_files",
			Description: "Browse the indexed file structure",
			Annotations: toolAnnotations(),
		}
	case "status":
		return &mcp.Tool{
			Name:        "codegraph_status",
			Description: "Report index health and counts",
			Annotations: toolAnnotations(),
		}
	default:
		panic("mcp: companionTool: unknown tool name " + name)
	}
}

// companionHandler registers one of the 7 companion tools on s via
// mcp.AddTool[In, any] (D-07). Every branch follows the same shape: build
// the typed args struct's engine call, open a fresh engine snapshot
// (openEngine), delegate to the matching Engine method, render via the
// matching internal/query Render*Markdown formatter (D-08b — no rendering
// re-implemented here), and return the markdown text. Panics on an
// unrecognized name — callers only ever pass names from companionNames.
//
// The switch dispatches on name here (rather than companionHandler
// returning a value for the caller to register) because Go generics
// require In to be resolved at compile time per call site — each branch's
// *Args type differs, so mcp.AddTool[In, any] must be instantiated inside
// the branch that knows its concrete In type, not by a caller holding an
// `any`-boxed handler it would otherwise have to re-switch on to unbox
// (02-CONTEXT.md's "prefer the cheap mechanism" calibration note: one
// switch, not two).
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
// paths return a bare error BEFORE reaching that point, and toolForErr
// converts it to an IsError:true result, so "no-op on isError results"
// holds structurally, with no redundant isError check needed. "status" is
// deliberately excluded: query.RenderStatusMarkdown already embeds its
// own verbose blockquote warning (D-17), and a second compact prefix
// would duplicate it. repoPath is the confinement root; defaultPath is
// the caller's start path (CR-01) — see BuildServer's doc comment for why
// they must stay distinct.
//
// WR-04 (02-REVIEW-2.md): parameter order is (s, name, repoPath,
// defaultPath), matching BuildServer's own (repoPath, startPath) declared
// order — see exploreHandler's doc comment for the full rationale.
//
// SDK-04: every required-argument validation failure (a missing "symbol"
// or "query") never reaches these handlers at all — go-sdk's AddTool
// validates arguments against the matching *Args struct's inferred schema
// before any handler body runs (02-RESEARCH.md Q5), so the
// RequireString-style checks these handlers used to open with are gone;
// every remaining error path simply returns the Go error, which
// toolForErr converts into the same IsError:true/Content shape
// mcp.NewToolResultError used to build by hand.
func companionHandler(s *mcp.Server, name, repoPath, defaultPath string, detector *gitmeta.CachingDetector) {
	tool := companionTool(name)
	switch name {
	case "node":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args NodeArgs) (*mcp.CallToolResult, any, error) {
			// NODE-03: 0 (the zero value when omitted) means "no line
			// hint" — not a valid 1-indexed source line — matching
			// Engine.Node's own nil-means-unset convention (EXPL-05/
			// NODE-04: keeps this handler a thin delegation to the same
			// Engine the CLI uses).
			var lineHint *int
			if args.Line != 0 {
				lineHint = &args.Line
			}

			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			out, err := eng.Node(args.Symbol, args.File, lineHint)
			if err != nil {
				return nil, nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "search":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, any, error) {
			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			locs, err := eng.Search(args.Query, args.Kind, args.Limit)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderSearchMarkdown(args.Query, locs)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "callers":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args CallersArgs) (*mcp.CallToolResult, any, error) {
			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			result, err := eng.Callers(args.Symbol, args.Limit)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderCallersMarkdown(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "callees":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args CalleesArgs) (*mcp.CallToolResult, any, error) {
			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			result, err := eng.Callees(args.Symbol, args.Limit)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderCalleesMarkdown(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "impact":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args ImpactArgs) (*mcp.CallToolResult, any, error) {
			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			result, err := eng.Impact(args.Symbol, args.Depth)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderImpactMarkdown(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "files":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args FilesArgs) (*mcp.CallToolResult, any, error) {
			opts := query.FilesOptions{
				Pattern: args.Pattern,
				Filter:  args.Filter,
				Depth:   args.Depth,
				Format:  args.Format,
			}

			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			result, err := eng.Files(opts)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderFilesMarkdown(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: query.WorktreeNotice(eng.WorktreeMismatch(ctx)) + out}},
			}, nil, nil
		})
	case "status":
		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args StatusArgs) (*mcp.CallToolResult, any, error) {
			eng, close, err := openEngine(args.Path, defaultPath, repoPath, detector)
			if err != nil {
				return nil, nil, err
			}
			defer close()

			// codegraph_status is EXCLUDED from the compact notice
			// (WORK-02/D-12): Engine.Status() already computes
			// StatusResult.WorktreeMismatch, and RenderStatusMarkdown
			// embeds it as its own verbose blockquote — mirroring TS's
			// withWorktreeNotice, which excludes codegraph_status for
			// exactly this reason (it carries its own verbose form
			// instead of the compact one).
			result, err := eng.Status(ctx)
			if err != nil {
				return nil, nil, err
			}
			out := query.RenderStatusMarkdown(result)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: out}},
			}, nil, nil
		})
	default:
		panic("mcp: companionHandler: unknown tool name " + name)
	}
}
