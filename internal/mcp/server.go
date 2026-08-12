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
	"github.com/seanb4t/codegraph-go/internal/query"
)

// version is this server's reported MCP implementation version. There is
// no project release-version concept yet (Phase 6 territory); a literal
// placeholder is fine here since MCP clients don't gate behavior on it.
const version = "0.1.0"

// instructions is SPEC-07's entire deliverable (03-05-PLAN Task 1):
// ServerOptions.Instructions below reaches BOTH the classic `initialize`
// result AND the `server/discover` result through the identical SDK field
// — deliberate, not incidental, because zero of the eight roster agent
// clients speak 2026-07-28 today, so a discover-only string would reach no
// real user. It ships to every client on every session, which makes it a
// wire contract: it MUST stay a compile-time literal with no interpolation
// of any kind — no repository path, no resolved index root, no hostname,
// no environment value — since a per-transcript variation here would
// publish the capturing host's filesystem layout into every committed
// wire-oracle transcript that carries it (T-03-19). Kept to a single
// paragraph with no newline characters and roughly 600 bytes or fewer,
// since the value is JSON-encoded into every one of those transcripts and
// an embedded newline would become an escape sequence that makes each such
// diff harder to read.
const instructions = "codegraph indexes this repository's code into a call and symbol graph; try codegraph_explore first for a where-is-X or how-does-Y-work question, since it returns verbatim source plus call paths in one call. Every tool accepts an optional path argument; omitting it uses this server's own working directory. All eight tools register by default and appear automatically once an index exists, with no client restart required; an empty tool list means this repository has no index yet, so run codegraph init. Setting CODEGRAPH_MCP_TOOLS narrows the surface to the companions it names."

// companionNames is the fixed vocabulary of the 7 tools CODEGRAPH_MCP_TOOLS
// may narrow the surface to — codegraph_explore is not in this list because
// it is always visible when hasIndex is true and is never removable by the
// filter.
var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}

// ResolveCompanions decides which of the 7 companion tools register,
// from the raw CODEGRAPH_MCP_TOOLS value and whether that variable was SET
// AT ALL. It is the ONE place the default lives; BuildServer knows only
// "here is the companion set", never how it was derived.
//
// present is load-bearing and MUST come from os.LookupEnv — never from a
// `value != ""` test. Under narrowing semantics "unset" and "set to the
// empty string" are two DIFFERENT answers (all 7 companions vs none), and
// os.Getenv collapses them into the same empty string. That collapse is the
// single most fragile point of this inversion, which is why the distinction
// is a parameter rather than an inference.
//
// Semantics (superseding the pre-inversion opt-in allowlist, which made
// codegraph_explore the only default-visible tool and required the operator
// to name every companion they wanted):
//
//   - unset            => all 7 companions register (8 tools total)
//   - "node,status"    => only those 2 register (3 tools total)
//   - ""               => no companion registers (explore only) — the
//     pre-inversion default, still reachable, now by
//     explicit operator action rather than by accident
//
// An index must still resolve for ANY tool to register; nothing here
// overrides MCP-03's zero-tools-without-.codegraph/ rule, which BuildServer
// enforces via hasIndex.
func ResolveCompanions(value string, present bool) (companions map[string]bool, unknown []string) {
	if !present {
		companions = make(map[string]bool, len(companionNames))
		for _, name := range companionNames {
			companions[name] = true
		}
		return companions, nil
	}
	return ParseToolFilter(value)
}

// ParseToolFilter splits a SET CODEGRAPH_MCP_TOOLS value on commas, trims
// whitespace around each entry, and classifies each non-empty name against
// companionNames. Recognized names are returned in selected
// (selected[name] == true); unrecognized names are returned in unknown, in
// the order they were seen, for the caller to warn about via
// WarnToolFilterTo — ParseToolFilter itself never writes output or aborts,
// so an unknown name can never fail startup (MCP-02: "unknown names ignored
// with a stderr warning").
//
// Callers should not reach for this directly: ResolveCompanions is the seam
// that knows an UNSET variable means something entirely different from an
// empty one. This function only ever sees the set case.
func ParseToolFilter(value string) (selected map[string]bool, unknown []string) {
	selected = make(map[string]bool)
	known := make(map[string]bool, len(companionNames))
	for _, n := range companionNames {
		known[n] = true
	}

	for _, raw := range strings.Split(value, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if known[name] {
			selected[name] = true
			continue
		}
		unknown = append(unknown, name)
	}
	return selected, unknown
}

// WarnToolFilterTo writes one stderr-style warning line per unrecognized
// filter name to w, followed by ONE line stating the resulting surface.
// Diagnostics never go to stdout — stdout is reserved for the MCP JSON-RPC
// transport (T-03-07-Leak) — so callers must pass os.Stderr, never
// os.Stdout, in production.
//
// The consequence line is not decoration. Under the pre-inversion opt-in
// allowlist a typo cost exactly the mistyped tool: the operator was ADDING,
// so the loss showed up as "the thing I asked for is missing", blast radius
// one. Under narrowing semantics a value whose names are ALL typos narrows
// the surface to codegraph_explore alone — blast radius seven, presenting as
// the exact "the MCP server is only showing one tool" symptom this contract
// was rewritten to eliminate. Naming what was ignored without naming what
// survived leaves the operator to work that out themselves.
//
// Deliberately NOT fatal: MCP-02 requires unknown names be ignored with a
// warning, and failing startup over one typo would take down a working
// server. Equally deliberately NOT a silent fallback to the default set —
// that would make a fully-typo'd value indistinguishable from a considered
// choice to narrow.
func WarnToolFilterTo(w io.Writer, unknown []string, companions map[string]bool) {
	if len(unknown) == 0 {
		return
	}
	for _, name := range unknown {
		fmt.Fprintf(w, "codegraph mcp: unknown tool name %q in CODEGRAPH_MCP_TOOLS, ignoring\n", name)
	}
	fmt.Fprintf(w, "codegraph mcp: CODEGRAPH_MCP_TOOLS narrows the tool surface to the names it lists; after ignoring %d unrecognized name(s), %d of %d companion tools will register alongside codegraph_explore. Unset the variable entirely to register all %d tools.\n",
		len(unknown), len(companions), len(companionNames), len(companionNames)+1)
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
func NewStdioServer(hasIndex bool, companions map[string]bool, repoPath, startPath string, sessionLog io.Writer) Server {
	if sessionLog == nil {
		panic("mcp.NewStdioServer: sessionLog must not be nil — pass io.Discard to explicitly opt out of the always-on VRFY-03 session line")
	}
	s := BuildServer(hasIndex, companions, repoPath, startPath, WithSessionLog(sessionLog))
	return &goSDKServer{inner: s}
}

// allToolNames returns codegraph_explore's name plus all 7 companion tool
// names (regardless of the filter), derived from exploreTool() and
// companionTool(name) rather than re-typed string literals — the single
// source registerTools and unregisterTools both read from, so the set a
// re-check registers and the set it unregisters can never drift apart.
// unregisterTools passes this whole set to (*mcp.Server).RemoveTools
// unconditionally; RemoveTools is documented as a no-op for a name that
// was never registered (e.g. a companion a narrowing filter excluded),
// so removing more names than were ever added is always safe.
func allToolNames() []string {
	names := make([]string, 0, 1+len(companionNames))
	names = append(names, exploreTool().Name)
	for _, name := range companionNames {
		names = append(names, companionTool(name).Name)
	}
	return names
}

// registerTools registers codegraph_explore plus every selected companion
// tool against s and returns the number of tools registered.
// This is SPEC-05's single registration seam (D-05): BuildServer's
// construction-time call and the per-request re-check inside
// AddReceivingMiddleware both go through this exact function, so
// "what the catalog contains" can never diverge between the two call
// sites. detector is the one gitmeta.CachingDetector constructed per
// server (D-13) — this function never constructs its own.
//
// Iteration order is companionNames', never the map's — a Go map range is
// randomized per run, and this function's call order is what determines the
// registration order a tools/list response echoes back.
func registerTools(s *mcp.Server, companions map[string]bool, repoPath, startPath string, detector *gitmeta.CachingDetector) int {
	count := 0

	// WR-04 (02-REVIEW-2.md): exploreHandler/companionHandler's parameter
	// order is (repoPath, startPath), matching BuildServer's own
	// (repoPath, startPath) signature.
	mcp.AddTool(s, exploreTool(), exploreHandler(repoPath, startPath, detector))
	count++
	for _, name := range companionNames {
		if companions[name] {
			companionHandler(s, name, repoPath, startPath, detector)
			count++
		}
	}
	return count
}

// unregisterTools is registerTools' mirror: it removes every tool name
// allToolNames() could ever have registered, via
// (*mcp.Server).RemoveTools. Safe to call even when only a subset (or
// none) of those names is currently registered — RemoveTools no-ops on a
// name that isn't present.
func unregisterTools(s *mcp.Server) {
	s.RemoveTools(allToolNames()...)
}

// BuildServer constructs the stdio MCP server with startup-time
// conditional tool registration (D-08a, Pattern 3): hasIndex gates
// whether ANY tool is registered at all (MCP-03 — zero tools when no
// .codegraph/ resolves, though MCP init still completes successfully),
// companions names which of the 7 companion tools register beyond the
// always-visible codegraph_explore.
//
// companions is a RESOLVED set, never a raw environment value: this
// function has no opinion about defaults, and in particular does not treat
// an empty map as "the caller didn't say, so give them everything."
// ResolveCompanions owns that decision, and an empty map here means exactly
// what it says — register codegraph_explore and nothing else.
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
func BuildServer(hasIndex bool, companions map[string]bool, repoPath, startPath string, opts ...Option) *mcp.Server {
	cfg := &buildConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// toolCount is derived at the registration seam (registerTools'
	// return value), never recomputed independently from
	// hasIndex/companions — an independent recomputation would be
	// duplicated state that silently drifts the first time a registration
	// condition changes. The !hasIndex case leaves it zero by
	// construction (the registerTools call below is skipped entirely).
	//
	// Phase 3/SPEC-05: this is now written from TWO places — this
	// function's construction-time registration below, and the
	// per-request re-check inside AddReceivingMiddleware further down —
	// and read from the session-line branch of that same middleware,
	// potentially from a different goroutine than either writer. A plain
	// int (its Phase-1/2 shape) is no longer safe for that access
	// pattern; atomic.Int64 makes "no store is ever observed torn or
	// reordered relative to a subsequent Load" a construction property
	// rather than a data race the race detector would need to catch by
	// luck. go-sdk exposes no tool-count accessor on *mcp.Server (no
	// Server.Tools() or count method — Server.Sessions() is the only
	// public feature-count-adjacent accessor, 03-RESEARCH.md Pitfall 2),
	// so this counter remains the only source of truth and must be kept
	// in sync at the exact point registration changes — both here and in
	// the re-check below.
	var toolCount atomic.Int64

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
			// RSRC-03/D-11 extension: explicit zero value, not omission.
			// go-sdk's capabilities() (server.go:645-653) would otherwise
			// auto-populate caps.Resources with ListChanged: true purely
			// because s.resources.len() > 0 once registerResources below
			// runs — this phase implements neither listChanged nor
			// subscribe, so the explicit zero value is what keeps the
			// advertised capability truthful, mirroring D-11's own
			// rationale for Tools above.
			Resources: &mcp.ResourceCapabilities{},
			Tools:     &mcp.ToolCapabilities{ListChanged: true},
		},
		Instructions: instructions,
	})

	// RSRC-03: registerResources runs unconditionally, immediately after
	// construction and BEFORE the `if hasIndex {` tool-registration branch
	// below — this call site's position outside that branch is the
	// structural property that makes resources available even in an
	// unindexed repository, mirroring how Capabilities.Tools above is set
	// regardless of hasIndex.
	registerResources(s)

	// One gitmeta.CachingDetector per SERVER, not per handler or per call
	// (D-13, corrected). openEngine builds a FRESH query.Engine on every
	// single tool call by design (its own doc comment says so), so an
	// Engine-scoped cache alone would give ZERO cross-call benefit on
	// this server's long-lived process — the exact surface the cache
	// exists to help. Detection costs up to four git subprocesses per
	// verdict; constructing exactly one detector here and closing it
	// over every handler — including any handler registered later by
	// the SPEC-05 re-check below — bounds that cost to once per
	// (startPath, indexRoot) pair for this server's entire lifetime,
	// however many tool calls follow. Phase 3/SPEC-05 moved this
	// construction outside the `if hasIndex` block that used to gate it:
	// the detector must exist even when hasIndex starts false, since the
	// re-check may call registerTools (and therefore need it) later in
	// this same server's lifetime — it is still constructed exactly
	// once, never per re-check.
	detector := gitmeta.NewCachingDetector()

	// catalogMu guards hasCatalog and every registerTools/unregisterTools
	// call made after construction (the per-request re-check below) —
	// the construction-time call just below does not need it (Run has
	// not been called yet, so no concurrent request can be in flight).
	// hasCatalog tracks this server's last-observed index presence, so a
	// re-check that finds no change makes no registration call at all
	// (TestRepeatedListsDoNotDuplicateTools' flip-guard).
	var catalogMu sync.Mutex
	hasCatalog := hasIndex

	if hasIndex {
		toolCount.Store(int64(registerTools(s, companions, repoPath, startPath, detector)))
	}

	// recheckCatalog is SPEC-05's per-request trigger (D-05): it
	// re-resolves whether an index exists at the server's
	// CONSTRUCTION-TIME startPath — never a request argument, see the
	// middleware call site below — and, on a state flip, calls
	// registerTools/unregisterTools (the exact same seam construction
	// used) so the live catalog can never diverge from what a fresh
	// BuildServer call would have produced for the same on-disk state.
	//
	// Mechanism: D-05 chose Server.AddTool/RemoveTools mutation over
	// per-call filtering because mutation is faithful to "the catalog
	// changed" and pre-builds most of SPEC-09's substrate (Phase 5's
	// subscriptions/listen work) — changeAndNotify (called by both
	// AddTool and RemoveTools) unconditionally emits
	// notifications/tools/list_changed to every Legacy session the
	// moment registration changes, a free, in-scope improvement this
	// plan writes no code for. Modern sessions receive that notification
	// only with an active subscriptions/listen stream, which Phase 5
	// builds — so this does NOT pull SPEC-09 forward.
	//
	// Trigger: internal/watch's fsnotify watcher is deliberately NOT
	// this mechanism. internal/cli/serve.go:71-78's own doc comment
	// states verbatim that an index created mid-session "is served live
	// by per-call query resolution but does NOT retroactively start the
	// watcher" (IN-09, a deliberate v1.0-era decision boundary this
	// phase does not move) — so a per-request re-check is SPEC-05's only
	// correct trigger, matching its own wording ("re-checked per call").
	//
	// Concurrency: AddTool/RemoveTools are s.mu-guarded through
	// changeAndNotify, and every reader a concurrent tools/call or
	// tools/list dispatches through (listTools, getServerTool,
	// capabilities()) acquires and releases that same s.mu briefly,
	// never across handler execution — calling them from inside this
	// middleware closure cannot deadlock and cannot race a concurrently
	// executing tool handler (03-RESEARCH.md § Pattern 2, source-traced;
	// exercised, not merely asserted, by Task 1's -race verify).
	//
	// Confinement: repoPath/startPath passed to registerTools here are
	// ALWAYS this closure's own construction-time values, never
	// re-derived from the resolved directory ResolveCodegraphDir
	// returns — an index appearing at or above startPath tightens
	// nothing about where tools may read from; confineToRepoRoot's
	// anchor (repoPath) is not moved. Widening it in response to
	// filesystem state a client may be able to influence would loosen
	// CR-02's trust boundary at runtime (T-03-14).
	recheckCatalog := func() {
		catalogMu.Lock()
		defer catalogMu.Unlock()

		_, err := query.ResolveCodegraphDir(startPath)
		switch {
		case err == nil:
			if !hasCatalog {
				toolCount.Store(int64(registerTools(s, companions, repoPath, startPath, detector)))
				hasCatalog = true
			}
		case errors.Is(err, query.ErrNotInitialized):
			if hasCatalog {
				unregisterTools(s)
				toolCount.Store(0)
				hasCatalog = false
			}
		default:
			// A transient stat failure is "state unknown, leave the
			// catalog as it is" — never treated as ErrNotInitialized,
			// which would silently empty a working catalog on a passing
			// filesystem hiccup.
		}
	}

	// The session-line + cacheScope middleware is registered
	// unconditionally, whether or not cfg.sessionLog is nil — only the
	// session-line WRITE is guarded on cfg.sessionLog != nil, so D-09's
	// cacheScope correction is never accidentally coupled to session
	// logging. Registered after the tool-registration block above so
	// toolCount's initial value is set by the time any request is
	// actually handled (Run has not been called yet at this point in any
	// caller, so this ordering is for readability, not correctness of
	// the closure capture).
	var mu sync.Mutex
	s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			// SPEC-05: re-check BEFORE next(...) runs, so the very same
			// request that observes a just-appeared (or just-removed)
			// index already reflects it in its own answer — no restart,
			// no reconnect, no "one call behind." Gated to the four
			// methods whose answer actually depends on the tool catalog;
			// running this on every notification (progress, cancelled,
			// etc.) would be an unbounded-per-request cost for no
			// observable benefit (T-03-15).
			switch method {
			case "initialize", "tools/list", "tools/call", "server/discover":
				recheckCatalog()
			}

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
				//
				// VRFY-03/T-03-17 (Phase 3): tools=N is the tool count
				// OBSERVED FOR THIS REQUEST — the recheckCatalog() call
				// above already ran before next(), so toolCount.Load()
				// here reads a live value, not a construction-time
				// constant. A user diagnosing "my tools vanished" reads
				// this line, and once the catalog is dynamic, tools=N no
				// longer describes the session's whole lifetime, only
				// the instant this initialize was answered — a
				// repudiation-adjacent semantics change, mitigated by
				// stating it here rather than leaving it implicit. The
				// "codegraph: mcp-session" prefix and every existing key
				// name (requested, negotiated, client, tools) are
				// UNCHANGED — Phase 1 D-16's one-way additive-only
				// contract holds; only what the tools value MEANS changed,
				// not its key name or the line's shape.
				mu.Lock()
				fmt.Fprint(cfg.sessionLog, formatSessionLine(
					params.ProtocolVersion,
					initRes.ProtocolVersion,
					params.ClientInfo.Name,
					params.ClientInfo.Version,
					int(toolCount.Load()),
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
			case "server/discover":
				// D-03: this closes SPEC-04's remaining discover half
				// — the identical defect D-09 fixed for tools/list
				// above, in the one response path Phase 2 never
				// touched. DiscoverResult embeds the same Cacheable
				// struct ListToolsResult does, and Server.discover
				// calls the same setDefaultCacheableValues, which is
				// why the correction shape transfers verbatim.
				// codegraph's catalog depends on a local .codegraph/
				// index resolving, so a client honoring "public" could
				// serve one repository's catalog for another — the
				// reason ttlMs: 0 and cacheScope: "private" are locked
				// in STATE.md as two halves of one correctness
				// property rather than independent options. TTLMs is
				// left untouched — it is already zero and already
				// correct.
				if discoverRes, ok := res.(*mcp.DiscoverResult); ok {
					discoverRes.CacheScope = "private"
				}
			case "resources/list", "resources/read":
				// cacheScope for resources is a decision, resolved here
				// rather than defaulted (05-RESEARCH.md Open Question 1).
				// go-sdk's setDefaultCacheableValues (protocol.go:1195-1197)
				// writes "public" unconditionally, and this middleware's
				// method-literal switch never saw resources methods before
				// this case. "private" is chosen over the SDK default for
				// three reasons: it keeps every codegraph response on the
				// wire uniform, so a reviewer never has to remember which
				// method families differ; STATE.md's standing decision
				// holds ttlMs: 0 and cacheScope: "private" to be two
				// halves of one correctness property rather than
				// independent options, and resources inherit the ttlMs: 0
				// half already; and "public" is fail-open for a future
				// resource whose content is repository-dependent, whereas
				// "private" is fail-safe and costs one case. This extends
				// the same two in-code decisions the "tools/list" and
				// "server/discover" cases above already made — those are
				// in-code decision IDs in a different numbering space from
				// 05-CONTEXT.md's own D-NN decisions, hence naming this
				// file rather than a D-NN tag here.
				switch r := res.(type) {
				case *mcp.ListResourcesResult:
					r.CacheScope = "private"
				case *mcp.ReadResourceResult:
					r.CacheScope = "private"
				}
			}
			return res, err
		}
	})

	return s
}
