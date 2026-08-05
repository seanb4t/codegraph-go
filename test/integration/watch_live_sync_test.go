// WR-04 (03-REVIEW.md): the one test that exercises the feature this phase
// actually ships — WATCH-01's value proposition ("the graph auto-updates"):
// a file written while a `serve --mcp` session is LIVE must become visible
// to a subsequent codegraph_explore without any manual `codegraph sync`,
// and no tool call during the flush window may error. Rides the SAME
// subprocess harness substrate 03-04 built (TestMain's binPath,
// copyFixture, runBinary) — no second TestMain, no new dependency.
package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	codegraphmcp "github.com/seanb4t/codegraph-go/internal/mcp"
)

// newServeClientWithEnv mirrors worktree_notice_test.go's newServeClient
// but threads extra environment variables into the spawned `serve --mcp`
// subprocess (appended to os.Environ(), never replacing it) — this test
// needs CODEGRAPH_DEBOUNCE_MS lowered so the debounced flush fires within
// a CI-friendly window instead of the 2s production default.
func newServeClientWithEnv(t *testing.T, ctx context.Context, cwd string, env []string) *mcpclient.Client {
	t.Helper()

	c, err := mcpclient.NewStdioMCPClientWithOptions(binPath, env, []string{"serve", "--mcp"},
		transport.WithCommandFunc(func(ctx context.Context, command string, cmdEnv []string, args []string) (*exec.Cmd, error) {
			cmd := exec.CommandContext(ctx, command, args...)
			cmd.Dir = cwd
			cmd.Env = append(os.Environ(), cmdEnv...)
			return cmd, nil
		}),
	)
	if err != nil {
		t.Fatalf("NewStdioMCPClientWithOptions(serve --mcp, cwd=%s, env=%v): %v", cwd, env, err)
	}
	t.Cleanup(func() { _ = c.Close() })

	req := mcp.InitializeRequest{}
	// codegraphmcp.ProtocolVersion is the repo-owned literal — a
	// dependency bump must not move this value silently (VRFY-02).
	req.Params.ProtocolVersion = codegraphmcp.ProtocolVersion
	req.Params.ClientInfo = mcp.Implementation{Name: "codegraph-integration-test", Version: "0.0.0"}
	if _, err := c.Initialize(ctx, req); err != nil {
		t.Fatalf("Initialize (cwd=%s): %v", cwd, err)
	}
	return c
}

// TestLiveEditAutoSyncReachesExplore proves WATCH-01 end-to-end against the
// real spawned binary: with the watcher default-on (no --watch in argv), a
// NEW symbol file written into the watched tree while the MCP session is
// live must show up in a codegraph_explore payload within a bounded window
// — and every explore issued while polling must succeed (no IsError, no
// transport error).
//
// This is also CR-01's (03-REVIEW.md) regression anchor: the polling loop
// deliberately overlaps per-call query opens with the debounced flush's
// indexer.Sync store open, both of which contend on Pebble's exclusive
// directory LOCK. The edit is a BURST of new files (not one) so the
// resulting sync holds the store's LOCK for a wide, reliably-collidable
// window, and explores are streamed with only a tiny gap between calls. On
// unfixed CR-01 behavior this fails one of two ways: an explore landing
// inside the flush window returns a lock error (IsError → t.Fatal), or the
// flush's own store open loses the race to an in-flight explore, is never
// retried, and the new symbol never appears (poll deadline → t.Fatal).
// With the fix (bounded lock retries in graphstore.Open plus the daemon's
// bounded flush requeue) both directions converge and the test passes.
//
// The anchor file is periodically re-touched: the watcher starts OFF the
// handshake path (D-06), so the initial burst racing watch.Open's
// recursive registration could be missed entirely; periodic rewrites
// guarantee an event lands after the watcher is live without any
// sleep-and-hope coordination (each rewrite is content-identical, so once
// synced, Sync's stat-prefilter treats subsequent touches as cheap mtime
// refreshes). The re-touch cadence (500ms) is deliberately LONGER than the
// debounce window (100ms here) so the touches can never permanently re-arm
// the debouncer and starve the flush.
func TestLiveEditAutoSyncReachesExplore(t *testing.T) {
	main := copyFixture(t)
	if _, stderr, err := runBinary(t, main, nil, "init", main); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	c := newServeClientWithEnv(t, ctx, main, []string{"CODEGRAPH_DEBOUNCE_MS=100"})

	// The live edit: one burst of genuinely new exported symbols in an
	// existing package. Zeta is the polled anchor; the sibling files exist
	// to widen the sync's LOCK-held window (every one of them stays in the
	// added set until the first sync SUCCEEDS, so each attempt is a full
	// 26-file parse — the window never degrades to a cheap no-op while the
	// race is still being exercised).
	zetaSrc := []byte("package pkga\n\n// Zeta is a live-session addition written while serve --mcp was running.\nfunc Zeta() int {\n\treturn Alpha() + 1\n}\n")
	zetaPath := filepath.Join(main, "pkga", "zeta.go")
	if err := os.WriteFile(zetaPath, zetaSrc, 0o644); err != nil {
		t.Fatalf("writing live-edit file %s: %v", zetaPath, err)
	}
	for i := 0; i < 25; i++ {
		src := fmt.Sprintf("package pkga\n\n// LiveGen%02d pads the live-edit burst (see test doc comment).\nfunc LiveGen%02d() int {\n\treturn Alpha() + %d\n}\n", i, i, i)
		p := filepath.Join(main, "pkga", fmt.Sprintf("gen_live_%02d.go", i))
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatalf("writing live-edit burst file %s: %v", p, err)
		}
	}

	deadline := time.Now().Add(90 * time.Second)
	lastTouch := time.Now()
	var lastPayload string
	for time.Now().Before(deadline) {
		result, err := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "codegraph_explore",
				Arguments: map[string]any{"query": "Zeta"},
			},
		})
		if err != nil {
			t.Fatalf("codegraph_explore transport error during a live watch session: %v", err)
		}
		if result.IsError {
			// The CR-01 signature: an explore landing inside a flush window
			// hitting Pebble's held directory LOCK must never surface as a
			// tool-call error.
			t.Fatalf("codegraph_explore returned a tool error during a live watch session (CR-01 store-lock collision?): %+v", result)
		}
		lastPayload = resultText(t, result)
		if strings.Contains(lastPayload, "func Zeta") {
			return // the auto-synced graph served the new symbol's verbatim source
		}

		// Re-touch the anchor so an event lands even if the initial burst
		// raced the watcher's off-handshake-path startup (see doc comment) —
		// but never faster than the debounce window, so the flush is never
		// starved by its own trigger.
		if time.Since(lastTouch) >= 500*time.Millisecond {
			if err := os.WriteFile(zetaPath, zetaSrc, 0o644); err != nil {
				t.Fatalf("re-touching live-edit file %s: %v", zetaPath, err)
			}
			lastTouch = time.Now()
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("codegraph_explore never reflected the live edit (func Zeta) within the deadline — the default-on watcher did not auto-sync the graph (WATCH-01); last payload:\n%s", lastPayload)
}
