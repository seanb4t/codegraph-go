// Package main implements tools/mcpaudit, the VRFY-05 proxying MCP capture
// shim: a standalone program a developer's real agent client (Claude Code,
// Codex CLI, opencode, ...) launches in place of `codegraph serve --mcp`.
// It observes — but never terminates or alters — the bidirectional JSON-RPC
// exchange between the agent and the real `codegraph` binary, so the
// dated 8-agent negotiation audit (docs/MCP-8-AGENT-AUDIT.md) can measure
// which protocolVersion each client actually negotiates on the wire,
// rather than reading it from documentation (D-09).
//
// Design constraints, carried over verbatim from CONTEXT.md and the phase
// threat model:
//
//   - The shim proxies, never terminates: the agent must keep working for
//     the whole audit (D-09, T-02-04). A parse failure is recorded in the
//     Observation's ParseError field and the proxy loop continues
//     unconditionally; aborting here would break the developer's live
//     agent session.
//   - Every byte is forwarded exactly as read in both directions —
//     including CRLF line endings and a final frame with no trailing
//     newline — via a byte-preserving bufio.Reader, never bufio.Scanner
//     (which strips terminators and would silently rewrite the wire,
//     T-02-03).
//   - Decoding never goes through an MCP SDK type (VRFY-01's discipline
//     applies to every measurement instrument in this milestone) — only
//     generic structs local to this package.
//   - A raw frame is never stored verbatim on parse failure (T-02-01):
//     only a SHA-256 digest plus a short, non-printable-escaped prefix.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// readerBufSize matches the wire oracle's own 10 MB stdout cap
// (test/integration/mcp_stdout_purity_test.go), so a single unusually
// large frame does not force extra reallocation churn in the common case.
const readerBufSize = 10 * 1024 * 1024

// frameDigestPrefixLen bounds how much of a malformed frame's raw bytes
// may appear (escaped, never verbatim-unescaped) inside a recorded
// ParseError — enough for a human to recognize what was wrong, never the
// whole frame (T-02-01).
const frameDigestPrefixLen = 64

// Observation is one completed — or, if the child process exited first,
// partial — measurement of a single shim invocation: the process-start
// metadata every invocation carries, plus whatever the initialize
// exchange revealed before it completed (or didn't). Exactly one
// Observation is appended to the log immediately at process start (the
// "process-start record", fields beyond PID/StartedAt/Argv still zero),
// and exactly one more is appended once the exchange completes or the
// child exits — so a client that spawns the server a second time is
// visible in the log as two independent invocations' worth of records
// (D-11's second-process-spawn signal).
type Observation struct {
	PID                       int             `json:"pid"`
	StartedAt                 string          `json:"startedAt"`
	Argv                      []string        `json:"argv"`
	FirstMethod               string          `json:"firstMethod,omitempty"`
	PreInitializeProbe        bool            `json:"preInitializeProbe"`
	InitializeID              json.RawMessage `json:"initializeId,omitempty"`
	OfferedProtocolVersion    string          `json:"offeredProtocolVersion,omitempty"`
	NegotiatedProtocolVersion string          `json:"negotiatedProtocolVersion,omitempty"`
	ClientName                string          `json:"clientName,omitempty"`
	ClientVersion             string          `json:"clientVersion,omitempty"`
	ServerName                string          `json:"serverName,omitempty"`
	ServerVersion             string          `json:"serverVersion,omitempty"`
	Capabilities              json.RawMessage `json:"capabilities,omitempty"`
	ParseError                string          `json:"parseError,omitempty"`
	FrameDigest               string          `json:"frameDigest,omitempty"`
}

// ClientFrame is the subset of a client->server JSON-RPC frame this shim
// records, decoded only through genericFrame and anonymous structs —
// never an MCP SDK type.
type ClientFrame struct {
	Method          string
	ID              json.RawMessage
	ProtocolVersion string
	ClientName      string
	ClientVersion   string
	Capabilities    json.RawMessage
}

// IsInitialize reports whether this frame is the client's initialize
// request.
func (f ClientFrame) IsInitialize() bool { return f.Method == "initialize" }

// ServerFrame is the subset of a server->client JSON-RPC frame this shim
// records.
type ServerFrame struct {
	ID              json.RawMessage
	IsResult        bool
	IsError         bool
	ProtocolVersion string
	ServerName      string
	ServerVersion   string
}

// genericFrame is the minimal JSON-RPC 2.0 shape every frame — request,
// notification, or response — can be decoded into without committing to
// which one it is. Nothing downstream of this decode step ever unmarshals
// into an SDK-owned type.
type genericFrame struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// ParseClientFrame decodes one client->server JSON-RPC line. It succeeds
// for any well-formed frame regardless of method — only empty input,
// invalid JSON, or a JSON object with neither a method nor a result/error
// key is rejected (the FailLoudly table TestParseFrameHelpersFailLoudly
// exercises). For an initialize request it also decodes protocolVersion,
// clientInfo, and capabilities out of params; for any other method those
// fields are left zero and only Method/ID are populated — the caller
// (sessionObserver) is what decides whether a non-initialize first frame
// counts as a pre-initialize probe (D-11), since that is session-level
// state ParseClientFrame itself has no way to know.
func ParseClientFrame(line []byte) (ClientFrame, error) {
	gf, err := decodeGenericFrame(line)
	if err != nil {
		return ClientFrame{}, fmt.Errorf("mcpaudit: ParseClientFrame: %w", err)
	}
	cf := ClientFrame{Method: gf.Method, ID: gf.ID}
	if gf.Method != "initialize" || len(gf.Params) == 0 {
		return cf, nil
	}
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
		ClientInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
		Capabilities json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(gf.Params, &params); err != nil {
		return cf, fmt.Errorf("mcpaudit: ParseClientFrame: initialize params did not decode: %w", err)
	}
	cf.ProtocolVersion = params.ProtocolVersion
	cf.ClientName = params.ClientInfo.Name
	cf.ClientVersion = params.ClientInfo.Version
	cf.Capabilities = params.Capabilities
	return cf, nil
}

// ParseServerFrame decodes one server->client JSON-RPC line, extracting
// the NEGOTIATED protocolVersion and serverInfo out of an initialize
// result when present — this is the only place the negotiated version can
// be measured; it is never inferred from what the client offered (D-11).
// Same FailLoudly discipline as ParseClientFrame.
func ParseServerFrame(line []byte) (ServerFrame, error) {
	gf, err := decodeGenericFrame(line)
	if err != nil {
		return ServerFrame{}, fmt.Errorf("mcpaudit: ParseServerFrame: %w", err)
	}
	sf := ServerFrame{ID: gf.ID}
	switch {
	case len(gf.Error) > 0:
		sf.IsError = true
		return sf, nil
	case len(gf.Result) > 0:
		sf.IsResult = true
		var result struct {
			ProtocolVersion string `json:"protocolVersion"`
			ServerInfo      struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		}
		if err := json.Unmarshal(gf.Result, &result); err != nil {
			return sf, fmt.Errorf("mcpaudit: ParseServerFrame: result did not decode: %w", err)
		}
		sf.ProtocolVersion = result.ProtocolVersion
		sf.ServerName = result.ServerInfo.Name
		sf.ServerVersion = result.ServerInfo.Version
		return sf, nil
	default:
		// A server-originated notification (method set, no result/error)
		// — not an error, just nothing this shim needs to record.
		return sf, nil
	}
}

// decodeGenericFrame is the shared FailLoudly gate both ParseClientFrame
// and ParseServerFrame delegate to: empty input, invalid JSON, and a JSON
// object with neither a method nor a result/error key are all rejected
// here, so both parse functions inherit identical failure behavior.
func decodeGenericFrame(line []byte) (genericFrame, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return genericFrame{}, errors.New("empty frame line")
	}
	var gf genericFrame
	if err := json.Unmarshal(trimmed, &gf); err != nil {
		return genericFrame{}, fmt.Errorf("frame is not valid JSON: %w", err)
	}
	if gf.Method == "" && len(gf.Result) == 0 && len(gf.Error) == 0 {
		return genericFrame{}, errors.New("frame has no method and no result/error key")
	}
	return gf, nil
}

// sessionObserver tracks one shim invocation's initialize exchange across
// both stdio directions until it completes (or the child process exits
// first), accumulating into a single Observation. Safe for concurrent use
// from the two forwarding goroutines Run starts.
type sessionObserver struct {
	mu        sync.Mutex
	base      Observation
	seenFirst bool
	seenInit  bool
	complete  bool
	initID    string
}

func newSessionObserver(base Observation) *sessionObserver {
	return &sessionObserver{base: base}
}

// isComplete reports whether both halves of the initialize exchange have
// already been recorded — once true, further lines are forwarded but no
// longer inspected (per the action's "stop observing" instruction).
func (o *sessionObserver) isComplete() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.complete
}

// observeClientLine inspects one client->server line. The first line ever
// seen sets FirstMethod and PreInitializeProbe (true iff that first line
// is not itself initialize); subsequent lines are still scanned for the
// initialize request itself, so a probe-then-initialize session yields
// both facts on the same Observation rather than trading one off against
// the other (D-11).
func (o *sessionObserver) observeClientLine(line []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.complete {
		return
	}
	cf, err := ParseClientFrame(line)
	if err != nil {
		o.recordParseErrorLocked(line, err)
		return
	}
	if !o.seenFirst {
		o.seenFirst = true
		o.base.FirstMethod = cf.Method
		o.base.PreInitializeProbe = !cf.IsInitialize()
	}
	if !o.seenInit && cf.IsInitialize() {
		o.seenInit = true
		o.base.InitializeID = cf.ID
		o.base.OfferedProtocolVersion = cf.ProtocolVersion
		o.base.ClientName = cf.ClientName
		o.base.ClientVersion = cf.ClientVersion
		o.base.Capabilities = cf.Capabilities
		o.initID = string(cf.ID)
	}
}

// observeServerLine inspects one server->client line. It only records
// anything once the client's initialize request has already been seen
// (so it has an ID to match against), and only completes the exchange
// when a result's id matches that recorded initialize ID.
func (o *sessionObserver) observeServerLine(line []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.complete || !o.seenInit {
		return
	}
	sf, err := ParseServerFrame(line)
	if err != nil {
		o.recordParseErrorLocked(line, err)
		return
	}
	if !sf.IsResult || o.initID == "" || string(sf.ID) != o.initID {
		return
	}
	o.base.NegotiatedProtocolVersion = sf.ProtocolVersion
	o.base.ServerName = sf.ServerName
	o.base.ServerVersion = sf.ServerVersion
	o.complete = true
}

// recordParseErrorLocked stores a parse failure without ever storing the
// offending frame verbatim: a SHA-256 digest of the whole line, plus the
// error text carrying only a short, non-printable-escaped prefix of the
// line's first frameDigestPrefixLen bytes (T-02-01). Callers must already
// hold o.mu.
func (o *sessionObserver) recordParseErrorLocked(line []byte, err error) {
	sum := sha256.Sum256(line)
	o.base.FrameDigest = hex.EncodeToString(sum[:])
	prefix := line
	if len(prefix) > frameDigestPrefixLen {
		prefix = prefix[:frameDigestPrefixLen]
	}
	o.base.ParseError = fmt.Sprintf("%s (first %d bytes: %q)", err.Error(), len(prefix), string(prefix))
}

// snapshot returns a copy of the Observation accumulated so far, safe to
// call at any point — including mid-exchange, for the partial-observation
// case where the child exits before the exchange completes.
func (o *sessionObserver) snapshot() Observation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.base
}

// appendObservation serializes obs as one JSON line and appends it to the
// log file at path, opened O_APPEND|O_CREATE|O_WRONLY with 0o600 so a
// second process spawn — or a second shim invocation entirely — adds
// records rather than truncating prior evidence, and so the log carries
// no more permissive an access mode than the client identity it records
// warrants (T-02-01).
func appendObservation(path string, obs Observation) error {
	b, err := json.Marshal(obs)
	if err != nil {
		return fmt.Errorf("mcpaudit: appendObservation: marshal: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("mcpaudit: appendObservation: open %s: %w", path, err)
	}
	defer f.Close()
	b = append(b, '\n')
	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("mcpaudit: appendObservation: write %s: %w", path, err)
	}
	return nil
}

// forwardLines copies src to dst using a byte-preserving bufio.Reader —
// ReadBytes('\n'), never a Scanner, which strips the line terminator and
// would silently rewrite CRLF input or invent a trailing newline on a
// final unterminated frame. Every line read (terminator included when
// present, omitted when the stream ended without one) is written to dst
// exactly as read, THEN handed to observe for side-effect-only
// inspection — so a slow or failing observe call never withholds bytes
// the other side is waiting on.
func forwardLines(src io.Reader, dst io.Writer, observe func(line []byte)) {
	r := bufio.NewReaderSize(src, readerBufSize)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			if _, werr := dst.Write(line); werr != nil {
				return
			}
			observe(line)
		}
		if err != nil {
			return // EOF or a genuine read error: nothing more to forward
		}
	}
}

// ExitCodeError signals that the real binary Run spawned exited with a
// non-zero status. main.go unwraps this via errors.As to set its own exit
// code without treating a normal non-zero child exit as a shim failure.
type ExitCodeError struct{ Code int }

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("mcpaudit: child process exited with status %d", e.Code)
}

// Config configures one Run invocation.
type Config struct {
	// RealBin is the absolute path to the real codegraph binary (or, in
	// tests, any stand-in binary) this shim proxies to.
	RealBin string
	// RealArgs are the arguments passed to RealBin — defaults to
	// []string{"serve", "--mcp"} at the flag-parsing layer in main.go.
	RealArgs []string
	// LogPath is the append-only observation log this invocation writes
	// to: one process-start record immediately, one completed-or-partial
	// Observation once the child exits or the exchange completes.
	LogPath string

	// In/Out/ErrOut are the shim's own stdio. They default to
	// os.Stdin/os.Stdout/os.Stderr when nil; tests override these to
	// script byte streams without a real terminal or pipe.
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

// Run proxies stdio between the caller (an agent client, or a test) and a
// spawned instance of cfg.RealBin, observing — but never terminating or
// altering — both directions until the initialize exchange completes,
// then continuing to forward transparently, unobserved, for the rest of
// the session. A parse failure is recorded and never aborts the proxy
// (T-02-04): the only errors Run itself returns are genuine setup/spawn
// failures, or an *ExitCodeError wrapping the child's own non-zero exit
// status so main.go can propagate it without treating it as a shim bug.
func Run(cfg Config) error {
	in := cfg.In
	if in == nil {
		in = os.Stdin
	}
	out := cfg.Out
	if out == nil {
		out = os.Stdout
	}
	errOut := cfg.ErrOut
	if errOut == nil {
		errOut = os.Stderr
	}

	base := Observation{
		PID:       os.Getpid(),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		Argv:      append([]string{cfg.RealBin}, cfg.RealArgs...),
	}
	if err := appendObservation(cfg.LogPath, base); err != nil {
		return err
	}
	obs := newSessionObserver(base)

	cmd := exec.Command(cfg.RealBin, cfg.RealArgs...)
	childIn, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcpaudit: Run: StdinPipe: %w", err)
	}
	childOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcpaudit: Run: StdoutPipe: %w", err)
	}
	cmd.Stderr = errOut

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mcpaudit: Run: start %s: %w", cfg.RealBin, err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer func() { _ = childIn.Close() }()
		forwardLines(in, childIn, func(line []byte) {
			if !obs.isComplete() {
				obs.observeClientLine(line)
			}
		})
	}()
	go func() {
		defer wg.Done()
		forwardLines(childOut, out, func(line []byte) {
			if !obs.isComplete() {
				obs.observeServerLine(line)
			}
		})
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	// Appended whether the exchange completed or not — an incomplete
	// measurement must read as incomplete, never as silently absent.
	if err := appendObservation(cfg.LogPath, obs.snapshot()); err != nil {
		return err
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			if code := exitErr.ExitCode(); code != 0 {
				return &ExitCodeError{Code: code}
			}
			return nil
		}
		return fmt.Errorf("mcpaudit: Run: wait: %w", waitErr)
	}
	return nil
}
