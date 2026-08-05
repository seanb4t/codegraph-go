// Package mcp implements the codegraph stdio MCP server (D-08): a
// startup-time conditional tool registration surface built on
// github.com/mark3labs/mcp-go. It imports only internal/query (the
// read-only engine + formatters) — never internal/graphstore's Pebble
// implementation directly — so the internal/graphstore/archtest boundary
// holds (D-08b). Every tool handler in tools.go delegates to the same
// internal/query.Engine methods and formatters the CLI uses, taking a
// fresh query.OpenAt snapshot per call (D-02/D-08b, RESEARCH Pitfall 2)
// — one engine, two front-ends, so MCP output shapes stay byte-identical
// to the CLI's without a second rendering path.
package mcp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// version is this server's reported MCP implementation version. There is
// no project release-version concept yet (Phase 6 territory); a literal
// placeholder is fine here since MCP clients don't gate behavior on it.
const version = "0.1.0"

// companionNames is the fixed vocabulary of the 7 tools CODEGRAPH_MCP_TOOLS
// may allowlist (D-08a, MCP-02) — codegraph_explore is not in this list
// because it is always visible when hasIndex is true (MCP-01) and is
// never gated by the allowlist.
var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}

// ParseAllowlist splits the CODEGRAPH_MCP_TOOLS env value on commas,
// trims whitespace around each entry, and classifies each non-empty
// name against companionNames (D-08a/MCP-02). Recognized names are
// returned in allowed (allowed[name] == true); unrecognized names are
// returned in unknown, in the order they were seen, for the caller to
// warn about via WarnUnknownToolsTo — ParseAllowlist itself never writes
// output or aborts, so an unknown name can never fail startup (MCP-02:
// "unknown names ignored with a stderr warning").
func ParseAllowlist(env string) (allowed map[string]bool, unknown []string) {
	allowed = make(map[string]bool)
	known := make(map[string]bool, len(companionNames))
	for _, n := range companionNames {
		known[n] = true
	}

	for _, raw := range strings.Split(env, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if known[name] {
			allowed[name] = true
			continue
		}
		unknown = append(unknown, name)
	}
	return allowed, unknown
}

// WarnUnknownToolsTo writes one stderr-style warning line per unknown
// allowlist name to w. Diagnostics never go to stdout — stdout is
// reserved for the MCP JSON-RPC transport (T-03-07-Leak) — so callers
// must pass os.Stderr, never os.Stdout, in production.
func WarnUnknownToolsTo(w io.Writer, unknown []string) {
	for _, name := range unknown {
		fmt.Fprintf(w, "codegraph mcp: unknown tool name %q in CODEGRAPH_MCP_TOOLS, ignoring\n", name)
	}
}

// Server is SDK-02's narrow seam: the entire surface internal/cli needs to
// bootstrap and run the stdio MCP server, with no SDK type anywhere in its
// signature. internal/cli/serve.go depends only on this interface (and
// NewStdioServer, below) — it never imports mark3labs/mcp-go/server.
type Server interface {
	ServeStdio() error
}

// mark3labsServer is Server's only implementation today: a thin adapter
// over the mark3labs/mcp-go-backed *server.MCPServer BuildServer
// constructs. A future SDK swap (Phase 2) adds a sibling implementation
// behind this same interface; internal/cli never needs to change.
type mark3labsServer struct{ inner *server.MCPServer }

func (m *mark3labsServer) ServeStdio() error { return server.ServeStdio(m.inner) }

// buildConfig holds BuildServer's optional configuration, set via Option
// values (the functional-options pattern) so every existing positional
// call site keeps compiling unchanged when a new option is added.
type buildConfig struct {
	// sessionLog, when non-nil, is where VRFY-03's always-on
	// "codegraph: mcp-session" line is written after every successful
	// initialize handshake. nil means the caller opted out entirely (only
	// legitimate today via BuildServer's own default — NewStdioServer,
	// this file's production entrypoint, refuses to construct a Server
	// with a nil session log at all).
	sessionLog io.Writer
}

// Option configures BuildServer via the functional-options pattern.
type Option func(*buildConfig)

// WithSessionLog sets the writer VRFY-03's always-on session line is
// written to. Passing a nil writer here is equivalent to omitting the
// option (no session line is emitted) — NewStdioServer is the seam that
// makes "always on" a construction guarantee rather than a convention; see
// its doc comment.
func WithSessionLog(w io.Writer) Option {
	return func(c *buildConfig) { c.sessionLog = w }
}

// NewStdioServer is internal/cli/serve.go's sole entrypoint into this
// package (SDK-02): it builds the server via BuildServer and returns it as
// the SDK-agnostic Server interface.
//
// sessionLog must not be nil, and NewStdioServer panics if it is.
// VRFY-03's always-on negotiated-version stderr line is the milestone's
// only mitigation for a spec-sanctioned silent version mismatch (Legacy
// mark3labs servers silently coerce an unrecognized protocolVersion rather
// than rejecting it) — so silently disabling that line by passing a nil
// writer through some future call site must be structurally impossible,
// not merely unlikely. Callers that genuinely want the line suppressed
// (there is no such production caller today) must pass io.Discard
// explicitly — a deliberate, greppable opt-out, never a nil default.
func NewStdioServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string, sessionLog io.Writer) Server {
	if sessionLog == nil {
		panic("mcp.NewStdioServer: sessionLog must not be nil — pass io.Discard to explicitly opt out of the always-on VRFY-03 session line")
	}
	s := BuildServer(hasIndex, allowlist, repoPath, startPath, WithSessionLog(sessionLog))
	return &mark3labsServer{inner: s}
}

// BuildServer constructs the stdio MCP server with startup-time
// conditional tool registration (D-08a, Pattern 3): hasIndex gates
// whether ANY tool is registered at all (MCP-03 — zero tools when no
// .codegraph/ resolves, though MCP init still completes successfully),
// allowlist gates which of the 7 companion tools register beyond the
// always-visible codegraph_explore (MCP-01/02).
//
// repoPath and startPath are DELIBERATELY DISTINCT (CR-01, the Phase-1
// CR-02 recurrence this parameter split fixes): repoPath is the
// confinement root — the RESOLVED index root every handler's
// confineToRepoRoot check anchors against, rejecting any client-supplied
// "path" argument that resolves outside it (CR-02/tools.go's trust
// boundary) — while startPath is the CALLER'S actual starting directory
// (serve.go's `start`, before ResolveCodegraphDir's upward walk), the
// value every handler falls back to when the caller omits "path" and the
// value that must reach query.OpenAt for WorktreeMismatch to have
// anything to compare. Because repoPath is always startPath itself or an
// ANCESTOR of it (it is ResolveCodegraphDir(startPath)'s own return
// value), confining the default startPath to repoPath always succeeds
// structurally — only an explicit, client-supplied "path" redirecting
// elsewhere can ever be rejected.
//
// opts is variadic (SDK-02/VRFY-03) specifically so every pre-existing
// positional call site (17 of them, all in tests, as of this change) keeps
// compiling unchanged — only NewStdioServer, this file's one production
// caller, passes WithSessionLog.
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string, opts ...Option) *server.MCPServer {
	cfg := &buildConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// toolCount is derived at the registration seam below (each AddTool
	// call increments it), never recomputed independently from
	// hasIndex/allowlist — an independent recomputation would be
	// duplicated state that silently drifts the first time a registration
	// condition changes. The !hasIndex early return leaves it zero by
	// construction.
	var toolCount int

	serverOpts := []server.ServerOption{server.WithToolCapabilities(true)}
	if cfg.sessionLog != nil {
		var mu sync.Mutex
		hooks := &server.Hooks{}
		hooks.AddAfterInitialize(func(_ context.Context, _ any, req *mcp.InitializeRequest, res *mcp.InitializeResult) {
			// A single fmt.Fprint call is not a formal atomicity
			// guarantee on an arbitrary io.Writer, and
			// AddAfterInitialize can fire more than once if a client
			// re-initializes — the mutex is what makes "never a
			// partially-written or interleaved session line" a
			// construction property rather than an assumption.
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprint(cfg.sessionLog, formatSessionLine(
				req.Params.ProtocolVersion,
				res.ProtocolVersion,
				req.Params.ClientInfo.Name,
				req.Params.ClientInfo.Version,
				toolCount,
			))
		})
		serverOpts = append(serverOpts, server.WithHooks(hooks))
	}

	s := server.NewMCPServer("codegraph", version, serverOpts...)
	if !hasIndex {
		return s
	}

	// One gitmeta.CachingDetector per SERVER, not per handler or per call
	// (D-13, corrected). openEngine builds a FRESH query.Engine on every
	// single tool call by design (its own doc comment says so), so an
	// Engine-scoped cache alone would give ZERO cross-call benefit on this
	// server's long-lived process — the exact surface the cache exists to
	// help. Detection costs up to four git subprocesses per verdict;
	// constructing exactly one detector here and closing it over every
	// handler bounds that cost to once per (startPath, indexRoot) pair for
	// this server's entire lifetime, however many tool calls follow.
	detector := gitmeta.NewCachingDetector()

	// WR-04 (02-REVIEW-2.md): exploreHandler/companionHandler's parameter
	// order is (repoPath, startPath), matching THIS function's own
	// (repoPath, startPath) signature above — before this fix, both calls
	// below inverted the order relative to BuildServer's own declared
	// params, a silent-swap footgun on two adjacent same-typed strings
	// that no test distinguished (see server_test.go's
	// TestConfinementAnchoredOnRepoRootNotStartPath, WR-02's companion
	// fix, for the test that now would catch a swap here).
	s.AddTool(exploreTool(), exploreHandler(repoPath, startPath, detector))
	toolCount++
	for _, name := range companionNames {
		if allowlist[name] {
			s.AddTool(companionTool(name), companionHandler(name, repoPath, startPath, detector))
			toolCount++
		}
	}
	return s
}
