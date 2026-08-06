// Package mcp implements the codegraph stdio MCP server (D-08): a
// startup-time conditional tool registration surface. It imports only
// internal/query (the read-only engine + formatters) — never
// internal/graphstore's Pebble implementation directly — so the
// internal/graphstore/archtest boundary holds (D-08b). Every tool handler in
// tools.go delegates to the same internal/query.Engine methods and
// formatters the CLI uses, taking a fresh query.OpenAt snapshot per call
// (D-02/D-08b, RESEARCH Pitfall 2) — one engine, two front-ends, so MCP
// output shapes stay byte-identical to the CLI's without a second rendering
// path.
//
// Phase 2 (SDK-01) migrated this package's backend from
// github.com/mark3labs/mcp-go to github.com/modelcontextprotocol/go-sdk —
// the official MCP org's SDK — behind the Server seam SDK-02 built.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
// NewStdioServer, below) — it never imports
// modelcontextprotocol/go-sdk/mcp.
type Server interface {
	ServeStdio() error
}

// goSDKServer is Server's implementation (Phase 2, SDK-01): a thin adapter
// over the modelcontextprotocol/go-sdk-backed *mcp.Server BuildServer
// constructs.
type goSDKServer struct{ inner *mcp.Server }

// ServeStdio runs the server over stdio.
//
// This is a deliberate DEVIATION from the plan's literal
// `g.inner.Run(context.Background(), &mcp.StdioTransport{})` — a
// load-bearing SDK-01 finding 02-RESEARCH.md's probe methodology did not
// surface. go-sdk dispatches every accepted request to a handler goroutine
// fully decoupled from its own stdin read loop
// ($GOMODCACHE/.../internal/jsonrpc2/conn.go's handleAsync). The instant
// that read loop's NEXT Read() call observes stdin EOF, the connection is
// marked "shutting down" and every subsequent Write — including the write
// of the response to the request already accepted and still being handled
// — is unconditionally rejected (conn.go's write()'s shuttingDown check;
// readIncoming's own exit path additionally cancels every still-in-flight
// incomingRequest, with the comment "the reader gone... so parked handlers
// have nothing useful left to do"). A client that writes its final request
// and closes stdin immediately afterward — exactly test/wireoracle's
// Capture() pattern, and an entirely ordinary batch-style MCP client shape
// — deterministically loses that response (confirmed empirically, 5/5
// runs against a real built binary; not a scheduler-dependent flake).
//
// Two mitigations were tried and rejected before this one, for the record:
//   - Delaying inside the handler does not help: the read loop's next
//     Read() call observes EOF independently of handler duration,
//     essentially immediately after the request is accepted.
//   - Tracking "requests accepted" via AddReceivingMiddleware and
//     decrementing when next() returns still loses the race
//     deterministically: middleware only runs once go-sdk's handleAsync
//     goroutine is actually scheduled, which is consistently SLOWER than
//     this process's own very next Read() call returning EOF from an
//     already-closed pipe (confirmed by instrumenting both sides: the
//     second Read() call observed a still-zero counter before the
//     goroutine had even started).
//
// The fix that works: this process owns the raw bytes crossing stdin, so
// it can synchronously recognize a complete JSON-RPC call line (one with
// both "method" and "id") the moment it is read — before returning those
// bytes to go-sdk's own decoder — and hold the pending count up from
// inside the SAME Read() call that produced them. That guarantees the
// increment happens-before any subsequent Read() call on the same
// reader can possibly observe EOF. stdinLingerReader does this via its
// own line buffering; pendingWriter decrements on the corresponding
// Write() actually reaching stdout, the only authoritative "this call's
// response left the process" signal. This is confined entirely to this
// method — no change to the Server interface, to BuildServer, to
// internal/cli/serve.go, or to any wire-visible response content; it
// only changes WHEN this process finally reports "no more input" once
// legitimately owed responses exist.
func (g *goSDKServer) ServeStdio() error {
	var pending atomic.Int64
	transport := &mcp.IOTransport{
		Reader: newStdinLingerReader(os.Stdin, &pending),
		Writer: &pendingWriter{w: os.Stdout, pending: &pending},
	}
	return g.inner.Run(context.Background(), transport)
}

// stdinLingerGrace bounds how long stdinLingerReader waits for accepted
// calls to finish writing their responses after observing stdin EOF,
// before giving up and propagating EOF to go-sdk's read loop anyway. This
// is a defensive upper bound, not an expected wait: normal request
// handling completes in low single-digit milliseconds against this
// server's local Pebble-backed engine, so the poll loop below almost
// always exits within a poll interval or two of pending reaching zero.
const stdinLingerGrace = 5 * time.Second

// stdinLingerPollInterval is how often stdinLingerReader re-checks the
// pending-response counter while waiting for it to drain.
const stdinLingerPollInterval = 2 * time.Millisecond

// stdinLingerReader wraps a stdin-shaped io.ReadCloser as a line-buffering
// proxy: it reads whole newline-delimited-JSON lines from the underlying
// reader (matching the exact framing go-sdk's own StdioTransport/
// IOTransport use), sniffs each line for a JSON-RPC call ("method" AND
// "id" both present — a notification carries "method" with no "id" and
// owes no response) to increment pending SYNCHRONOUSLY before the line's
// bytes are ever handed back to the caller, and — once the underlying
// reader reports real EOF — blocks (bounded by stdinLingerGrace) until
// pending drains to zero before propagating that EOF onward. See
// ServeStdio's doc comment for why the increment must happen here, not in
// a request-dispatch hook.
type stdinLingerReader struct {
	br      *bufio.Reader
	closer  io.Closer
	pending *atomic.Int64
	// unread holds the tail of a line already sniffed and buffered but not
	// yet copied out to a caller-supplied Read(p) slice.
	unread []byte
}

func newStdinLingerReader(r io.ReadCloser, pending *atomic.Int64) *stdinLingerReader {
	return &stdinLingerReader{br: bufio.NewReader(r), closer: r, pending: pending}
}

func (s *stdinLingerReader) Read(p []byte) (int, error) {
	if len(s.unread) == 0 {
		line, readErr := s.br.ReadBytes('\n')
		if len(line) > 0 {
			if looksLikeJSONRPCCall(line) {
				s.pending.Add(1)
			}
			s.unread = line
		}
		if readErr != nil && len(s.unread) == 0 {
			if errors.Is(readErr, io.EOF) {
				s.waitForDrain()
			}
			return 0, readErr
		}
		// A trailing partial line (readErr != nil but bytes were still
		// returned, e.g. the stream ended without a final newline) is
		// returned to the caller now; the error itself is reported on
		// the NEXT call once unread is drained, matching bufio.Reader's
		// own "data now, error later" contract — ReadBytes already
		// followed that contract for us, so nothing extra to do here.
	}
	n := copy(p, s.unread)
	s.unread = s.unread[n:]
	return n, nil
}

func (s *stdinLingerReader) waitForDrain() {
	deadline := time.Now().Add(stdinLingerGrace)
	for s.pending.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(stdinLingerPollInterval)
	}
}

func (s *stdinLingerReader) Close() error { return s.closer.Close() }

// sniffedMessage extracts just enough of a JSON-RPC line to classify it —
// never the full message, which go-sdk's own decoder owns.
type sniffedMessage struct {
	Method string           `json:"method"`
	ID     *json.RawMessage `json:"id"`
}

// looksLikeJSONRPCCall reports whether line is a JSON-RPC call (as opposed
// to a notification, which carries "method" but no "id" and receives no
// response). codegraph's stdio server never itself initiates client-bound
// requests, so every line read here originates as a client request or
// notification, never a response — no response-shaped line ever needs
// classifying. On a parse failure, this conservatively reports true: an
// unparseable line still gets SOME reply from go-sdk's own framing-error
// handling, and over-counting pending only ever costs a bounded wait
// against stdinLingerGrace, never incorrect behavior.
func looksLikeJSONRPCCall(line []byte) bool {
	var msg sniffedMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return true
	}
	return msg.Method != "" && msg.ID != nil
}

// pendingWriter wraps stdout and decrements pending on every observed
// Write() — the authoritative "a message actually reached the wire"
// signal stdinLingerReader waits on. See ServeStdio's doc comment.
type pendingWriter struct {
	w       io.Writer
	pending *atomic.Int64
}

func (p *pendingWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.pending.Add(-1)
	return n, err
}

// Close is a trivial no-op: IOTransport's underlying rwc.Close() closes
// both the Reader and the Writer, and os.Stdout must not actually be
// closed on shutdown (mirroring go-sdk's own unexported StdioTransport
// treatment of stdout).
func (p *pendingWriter) Close() error { return nil }

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
	return &goSDKServer{inner: s}
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
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string, opts ...Option) *mcp.Server {
	cfg := &buildConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// toolCount is derived at the registration seam below (each AddTool
	// call increments it), never recomputed independently from
	// hasIndex/allowlist — an independent recomputation would be
	// duplicated state that silently drifts the first time a registration
	// condition changes. The !hasIndex case leaves it zero by
	// construction (the registration block below is skipped entirely).
	var toolCount int

	// D-11: Capabilities must be set explicitly and unconditionally.
	// Server.capabilities() only sets caps.Tools when HasTools ||
	// tools.len() > 0 — without this, the "tools" key silently vanishes
	// from the initialize response's capabilities object on the
	// hasIndex=false path (MCP-03), which is exactly the ambiguous
	// "no tools = not indexed, or protocol mismatch?" failure mode this
	// milestone exists to close. Leaving Capabilities nil would also
	// inject an SDK-default "logging" capability codegraph does not
	// implement. HasTools is deliberately left unset — it is deprecated.
	s := mcp.NewServer(&mcp.Implementation{Name: "codegraph", Version: version}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{ListChanged: true},
		},
	})

	if hasIndex {
		// One gitmeta.CachingDetector per SERVER, not per handler or per
		// call (D-13, corrected). openEngine builds a FRESH query.Engine
		// on every single tool call by design (its own doc comment says
		// so), so an Engine-scoped cache alone would give ZERO cross-call
		// benefit on this server's long-lived process — the exact
		// surface the cache exists to help. Detection costs up to four
		// git subprocesses per verdict; constructing exactly one detector
		// here and closing it over every handler bounds that cost to
		// once per (startPath, indexRoot) pair for this server's entire
		// lifetime, however many tool calls follow.
		detector := gitmeta.NewCachingDetector()

		// WR-04 (02-REVIEW-2.md): exploreHandler/companionHandler's
		// parameter order is (repoPath, startPath), matching THIS
		// function's own (repoPath, startPath) signature above.
		mcp.AddTool(s, exploreTool(), exploreHandler(repoPath, startPath, detector))
		toolCount++
		for _, name := range companionNames {
			if allowlist[name] {
				companionHandler(s, name, repoPath, startPath, detector)
				toolCount++
			}
		}
	}

	// The session-line + cacheScope middleware is registered
	// unconditionally, whether or not cfg.sessionLog is nil — only the
	// session-line WRITE is guarded on cfg.sessionLog != nil, so D-09's
	// cacheScope correction is never accidentally coupled to session
	// logging. Registered after the tool-registration block above so
	// toolCount's final value is set by the time any request is actually
	// handled (Run has not been called yet at this point in any caller,
	// so this ordering is for readability, not correctness of the
	// closure capture).
	var mu sync.Mutex
	s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			res, err := next(ctx, method, req)
			if err != nil {
				return res, err
			}
			switch method {
			case "initialize":
				if cfg.sessionLog == nil {
					break
				}
				initRes, ok := res.(*mcp.InitializeResult)
				if !ok {
					break
				}
				params, ok := req.GetParams().(*mcp.InitializeParams)
				if !ok {
					break
				}
				// A single fmt.Fprint call is not a formal atomicity
				// guarantee on an arbitrary io.Writer, and this
				// middleware can fire more than once if a client
				// re-initializes — the mutex is what makes "never a
				// partially-written or interleaved session line" a
				// construction property rather than an assumption.
				//
				// The negotiated value is read off initRes, the SDK's
				// own computed *InitializeResult — never re-derived by
				// duplicating go-sdk's unexported negotiatedVersion
				// logic (02-RESEARCH.md § Don't Hand-Roll) — and the
				// requested value is read off params, the request the
				// SDK itself parsed, never off session state (which
				// only stores the request, never the negotiated result
				// — 02-RESEARCH.md Q1).
				mu.Lock()
				fmt.Fprint(cfg.sessionLog, formatSessionLine(
					params.ProtocolVersion,
					initRes.ProtocolVersion,
					params.ClientInfo.Name,
					params.ClientInfo.Version,
					toolCount,
				))
				mu.Unlock()
			case "tools/list":
				// D-09: go-sdk's listTools unconditionally calls
				// setDefaultCacheableValues, which writes CacheScope =
				// "public". codegraph's tool catalog depends on
				// whether a local .codegraph/ index resolves, so a
				// client honoring "public" could serve one repo's
				// catalog for another — correct it to "private" here.
				// TTLMs is left at its inherited zero, already the
				// correct target. Phase 3 owns SPEC-04's remaining
				// half (the per-call hasIndex re-check).
				if listRes, ok := res.(*mcp.ListToolsResult); ok {
					listRes.CacheScope = "private"
				}
			}
			return res, err
		}
	})

	return s
}
