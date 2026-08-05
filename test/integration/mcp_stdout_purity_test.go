// Task 1 (D-06a): the runtime half of HYG-02's stdout-purity guarantee.
// Plan 04-02 already proved the structural half (an archtest that fails
// the build if any serve-reachable package references os.Stdout / a bare
// fmt.Print* / log.SetOutput) — this file proves the same guarantee holds
// end-to-end, against the REAL binary's REAL stdout bytes.
package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/mcp"
)

// syncBuffer is a concurrency-safe io.Writer wrapping a bytes.Buffer. It
// exists because os/exec starts a background goroutine to copy a
// subprocess's stderr pipe into cmd.Stderr whenever that field is set to a
// non-*os.File io.Writer, and that goroutine is only joined by cmd.Wait()
// — which neither failure branch below calls before reading the buffer's
// contents. A plain bytes.Buffer would race under that goroutine's
// concurrent writes (WR-01); guarding both Write and String with a mutex
// makes the read/write safe regardless of ordering.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// TestServeMCPStdoutIsPureJSONRPC is D-06a's mandatory anchor: it spawns the
// real `serve --mcp` binary itself (plain exec.Command, NOT
// newServeClient/mcpclient.Client) and reads cmd.StdoutPipe() directly,
// asserting every single line is a valid JSON-RPC frame.
//
// This is deliberately NOT layered on the existing newServeClient helper.
// mcp-go@v0.56.0's stdio client transport
// (client/transport/stdio.go:readResponses) does
// `if err := json.Unmarshal(...); err != nil { continue }` on every stdout
// line — any non-JSON-RPC byte is silently skipped, never surfaced as an
// error, so a test built on Client.Initialize/Client.CallTool succeeding
// would pass even with garbage interleaved on stdout (04-RESEARCH.md
// Pitfall 1). Only a raw reader that owns the whole stream itself can
// actually fail on a real violation.
//
// The tools/call below names codegraph_status, deliberately exercising a
// SECOND store-open (openEngine) on top of the startup reconcile
// (indexer.Sync, which itself opens the store before ServeStdio ever
// starts reading stdin) — together the noise-provoking path D-06a
// requires. CODEGRAPH_MCP_TOOLS=status allowlists that companion tool
// (codegraph_explore is the only tool visible with no allowlist entry).
// CODEGRAPH_NO_WATCH=1 keeps the case deterministic: the watcher runs off
// the handshake path regardless (WATCH-02) and its diagnostics go to
// stderr, never stdout, so disabling it only removes timing
// nondeterminism, not anything this assertion depends on.
func TestServeMCPStdoutIsPureJSONRPC(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture: %v: %s", err, stderr)
	}

	cmd := exec.Command(binPath, "serve", "--mcp")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CODEGRAPH_MCP_TOOLS=status", "CODEGRAPH_NO_WATCH=1")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	stderrBuf := &syncBuffer{}
	cmd.Stderr = stderrBuf // diagnostics land here — never asserted for purity

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	writeLine := func(v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		b = append(b, '\n')
		if _, err := stdin.Write(b); err != nil {
			t.Fatalf("write request: %v", err)
		}
	}

	// Hand-framed initialize request (id 1) — the OS pipe buffers this
	// even if the subprocess is still mid-reconcile; ServeStdio consumes
	// it once it starts its read loop.
	writeLine(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			// mcp.ProtocolVersion is the repo-owned literal (this is the
			// only SDK-mcp reference this file had; the import above is
			// now internal/mcp, not the SDK's mcp package) — a dependency
			// bump must not move this value silently (VRFY-02).
			"protocolVersion": mcp.ProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "codegraph-purity-test",
				"version": "0.0.0",
			},
		},
	})
	// Hand-framed tools/call request (id 2) naming codegraph_status — the
	// store-opening call this test exists to provoke noise from. No
	// notifications/initialized round trip is required first: the server
	// marks its stdio session Initialized() synchronously inside
	// handleInitialize, before the initialize response is even written.
	writeLine(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "codegraph_status",
			"arguments": map[string]any{},
		},
	})

	type scannedLine struct {
		raw []byte
	}
	lines := make(chan scannedLine)
	// scanErr is written only by the scanning goroutine below, only before
	// close(lines) — the Go memory model guarantees a close(ch) happens
	// before a subsequent receive that returns because ch is closed, so
	// reading scanErr from the main goroutine's `!ok` branch below is safe
	// without additional synchronization.
	var scanErr error
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			b := append([]byte(nil), scanner.Bytes()...)
			lines <- scannedLine{raw: b}
		}
		scanErr = scanner.Err()
	}()

	var sawInitResponse, sawToolResponse bool
	deadline := time.After(30 * time.Second)
	for !sawInitResponse || !sawToolResponse {
		select {
		case ln, ok := <-lines:
			if !ok {
				// scanErr surfaces WHY the scan loop stopped early — a
				// nil error means clean EOF (subprocess exited/closed
				// stdout), but a non-nil error (e.g. bufio.ErrTooLong
				// from a runaway single line, or a pipe read error)
				// means the loop exited having silently swallowed
				// unverified stdout bytes; report it rather than only
				// saying "closed".
				t.Fatalf("stdout closed before both responses were seen (init=%v tool=%v); scanner error: %v; stderr:\n%s",
					sawInitResponse, sawToolResponse, scanErr, stderrBuf.String())
			}
			// EVERY line must parse as a JSON-RPC frame — this is the
			// purity assertion itself. A non-frame byte fails immediately
			// with the offending bytes quoted (acceptance criteria).
			var frame struct {
				JSONRPC string          `json:"jsonrpc"`
				ID      float64         `json:"id"`
				Error   json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(ln.raw, &frame); err != nil || frame.JSONRPC == "" {
				t.Fatalf("non-JSON-RPC byte on stdout: %q", ln.raw)
			}
			switch frame.ID {
			case 1:
				sawInitResponse = true
			case 2:
				// WR-02: a JSON-RPC error response (e.g. an allowlist
				// regression rejecting codegraph_status as unknown before
				// ever reaching graphstore.Open) would otherwise still
				// satisfy sawToolResponse here, silently no longer
				// exercising the second-store-open noise path this test
				// exists to provoke, while still reporting success.
				if len(frame.Error) > 0 {
					t.Fatalf("codegraph_status tools/call returned a JSON-RPC error, the store-open path was not exercised: %s", frame.Error)
				}
				sawToolResponse = true
			}
		case <-deadline:
			t.Fatalf("timed out waiting for initialize+tools/call responses (init=%v tool=%v)", sawInitResponse, sawToolResponse)
		}
	}
}
