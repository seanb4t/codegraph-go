// Task 1/2 (D-21): TEST-04's WATCH-coverage cases, riding the SAME
// subprocess harness substrate 03-04 built (binPath, buildWorktreeFixture,
// newServeClient) — no second TestMain, no new dependency.
package integration

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	codegraphmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// TestDefaultWatchHandshakePrompt proves WATCH-01/WATCH-02 end-to-end: a
// bare `serve --mcp` subprocess (NO --watch/--no-watch in argv — the
// default-on path, D-01/D-06) completes Initialize (via newServeClient) and
// ListTools within a single bounded context, and advertises
// codegraph_explore. This is a promptness/reachability assertion, not a
// timing microbenchmark — the bound just has to be comfortably longer than
// a healthy handshake and short enough that a watcher-startup stall (the
// WSL2 failure mode D-06 makes structurally impossible) would trip it.
//
// Reuses 03-04's buildWorktreeFixture (git-backed, t.Skip on git absence —
// the established runGitI convention) purely for its indexed main
// checkout; the worktree half of that fixture is unused here.
func TestDefaultWatchHandshakePrompt(t *testing.T) {
	_, main := buildWorktreeFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// newServeClient spawns `serve --mcp` with NO watch flag in argv (the
	// default-on path) and completes Initialize under ctx.
	c := newServeClient(t, ctx, main)

	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools on default-on serve --mcp: %v (default-on watcher must not delay the handshake or first-tool availability)", err)
	}

	found := false
	for _, tool := range result.Tools {
		if tool.Name == "codegraph_explore" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("codegraph_explore not advertised by default-on serve --mcp; got tools: %v", names)
	}
}

// TestNoWatchEnvDisablesViaStderr proves WATCH-03 end-to-end: with
// CODEGRAPH_NO_WATCH=1 in the spawned subprocess's environment, serve --mcp
// still completes Initialize (best-effort/never-block — disabling the
// watcher must not fail serve, D-06/serveWatchStart's ErrWatchDisabled
// branch), and the child's real stderr (via client.GetStderr, mcp-go's
// documented seam) carries the verbatim D-12/D-13 disabled message.
//
// The disabled message is printed by serve.go's watcher goroutine, which
// runs off the handshake path (D-06) — so it may land slightly after
// Initialize returns. Stderr is read continuously from a goroutine started
// BEFORE Initialize to avoid a race against the pipe buffer, and the
// assertion polls a bounded grace window rather than expecting the message
// strictly before Initialize.
func TestNoWatchEnvDisablesViaStderr(t *testing.T) {
	_, main := buildWorktreeFixture(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := mcpclient.NewStdioMCPClientWithOptions(binPath, []string{"CODEGRAPH_NO_WATCH=1"}, []string{"serve", "--mcp"},
		transport.WithCommandFunc(func(cctx context.Context, command string, env []string, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(cctx, command, args...)
			cmd.Dir = main
			cmd.Env = append(os.Environ(), env...)
			return cmd, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewStdioMCPClientWithOptions(serve --mcp, CODEGRAPH_NO_WATCH=1): %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	stderrReader, ok := mcpclient.GetStderr(c)
	if !ok {
		t.Fatal("client.GetStderr: transport does not expose the subprocess's stderr")
	}

	var mu sync.Mutex
	var stderrBuf bytes.Buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := stderrReader.Read(buf)
			if n > 0 {
				mu.Lock()
				stderrBuf.Write(buf[:n])
				mu.Unlock()
			}
			if readErr != nil {
				return
			}
		}
	}()

	req := mcp.InitializeRequest{}
	// codegraphmcp.ProtocolVersion is the repo-owned literal — a
	// dependency bump must not move this value silently (VRFY-02).
	req.Params.ProtocolVersion = codegraphmcp.ProtocolVersion
	req.Params.ClientInfo = mcp.Implementation{Name: "codegraph-integration-test", Version: "0.0.0"}
	if _, err := c.Initialize(ctx, req); err != nil {
		t.Fatalf("Initialize (CODEGRAPH_NO_WATCH=1): %v (disabling the watcher must never fail serve)", err)
	}

	// D-12/D-13 verbatim disabled message: "[CodeGraph MCP] File watcher
	// disabled — CODEGRAPH_NO_WATCH=1 is set. The graph will not
	// auto-update; run `codegraph sync` (or install the git sync hooks via
	// `codegraph init`) to refresh."
	const wantSubstring = "File watcher disabled — CODEGRAPH_NO_WATCH=1 is set"
	deadline := time.Now().Add(10 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		mu.Lock()
		got = stderrBuf.String()
		mu.Unlock()
		if strings.Contains(got, wantSubstring) {
			if !strings.Contains(got, "codegraph sync") {
				t.Fatalf("disabled stderr message missing the `codegraph sync` refresh guidance: %q", got)
			}
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("expected child stderr to contain %q within the grace window, got: %q", wantSubstring, got)
}
