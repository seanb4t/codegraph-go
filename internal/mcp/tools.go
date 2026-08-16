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

// toolAnnotations returns the ToolAnnotations every codegraph MCP tool
// carries: read-only, closed-world.
//
// These were inherited mark3labs zero-values (readOnlyHint:false,
// destructiveHint:true, idempotentHint:false, openWorldHint:true), carried
// through the go-sdk migration unexamined and deferred to "Phase 3" by a
// comment that outlived Phase 3. The deferral is discharged here. Until it
// was, `/mcp` in Claude Code rendered codegraph_explore — a graph query —
// as "destructive, open-world", the same risk vocabulary a host uses for a
// tool that deletes files and calls the internet.
//
// PER-TOOL AUDIT (all eight verified individually against their actual
// reachable call graph, not assumed from the tool's name):
//
//	codegraph_explore  -> Engine.Explore
//	codegraph_node     -> Engine.Node
//	codegraph_search   -> Engine.Search
//	codegraph_callers  -> Engine.Callers
//	codegraph_callees  -> Engine.Callees
//	codegraph_impact   -> Engine.Impact
//	codegraph_files    -> Engine.Files
//	codegraph_status   -> Engine.Status
//
// Every one of those eight reaches the store through openEngine, this
// file's single read seam, which hands back a query.Engine holding a
// graphstore.Reader — an interface with Get*/Iterate* methods and no
// mutator at all, so a write is not merely absent from these handlers, it
// is unreachable through the type they hold. No handler imports or calls
// internal/indexer, internal/daemon, or internal/watch, so none can trigger
// a reindex or touch the daemon lock. The startup reconcile and the file
// watcher that DO write are serve.go's, on the server's own lifecycle,
// never on a tool call.
//
// codegraph_status was scrutinized specifically, being the one whose name
// suggests it might refresh what it reports: it does not. It scans the
// snapshot (IterateFiles/IterateNodes/GetMeta), stats the store directory
// for a size, and compares source mtimes against the last sync stamp to
// decide whether to REPORT staleness. It reports; it never repairs.
//
// One honest caveat, deliberately not treated as disqualifying:
// graphstore.Open opens Pebble read-write, which takes the store's
// exclusive directory lock and can write WAL/manifest bytes under
// .codegraph/store/. That is storage-engine bookkeeping, invisible above
// the Reader interface and incapable of changing a single node, edge, or
// file record. readOnlyHint describes the tool's effect on its
// ENVIRONMENT — the graph, the repository, the host — and by that measure
// all eight are read-only. Reporting them as mutating because Pebble
// touched its own WAL would misinform a host about real risk.
//
// Field-by-field, against the spec's own definitions:
//
//   - ReadOnlyHint: true — none of the eight modifies its environment.
//   - DestructiveHint: nil (omitted) — "meaningful only when ReadOnlyHint
//     == false". Emitting it alongside readOnlyHint:true would be
//     incoherent in either direction, so it is left unset rather than set
//     to a value that cannot be true or false meaningfully.
//   - IdempotentHint: false — also meaningful only when ReadOnlyHint is
//     false. go-sdk serializes this field unconditionally (no omitempty on
//     a plain bool), so its zero value ships; that is inert, not a claim.
//   - OpenWorldHint: false — the domain of interaction is a local,
//     pre-built index. Nothing under internal/query or internal/gitmeta
//     imports net/http or dials anything; the only subprocesses in the
//     reachable set are four read-only `git rev-parse` calls used for
//     worktree detection, which are local and closed-world.
//
// Per the spec these remain HINTS, not authorization: a client must treat
// them as untrusted unless the server is trusted. Nothing here is a
// substitute for confineToRepoRoot above, which is the actual boundary.
func toolAnnotations() *mcp.ToolAnnotations {
	openWorld := false
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: nil,
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
// file, whose consumer is a language model, not a parser. The MCP tools
// render markdown because their consumer is a language model, while the
// CLI --json path marshals JSON because its consumer is a parser —
// SURF-06/D-16 made the two consistent with their consumers.
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
			// embeds it as its own verbose blockquote (D-17), so a second
			// compact prefix would duplicate it.
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
