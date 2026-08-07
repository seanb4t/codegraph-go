package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// TestReconnectReconcile pins D-06/SYNC-03: on MCP (re)connect, a reconcile
// pass (the exact indexer.Sync entry internal/cli/serve.go's RunE invokes
// before mcp.BuildServer — no second reconciliation code path, RESEARCH
// Pattern 5) catches an offline change made while no watcher/daemon was
// running, so the first codegraph_explore call after reconnect reads a
// current graph. A second reconcile with nothing changed must be a cheap
// no-op (zero files reparsed) — the stat pre-filter makes this true by
// construction.
func TestReconnectReconcile(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	// "Offline": mutate a fixture file on disk while no watcher/daemon
	// was running to reparse it live — the offline-change scenario D-06
	// exists to catch.
	target := filepath.Join(dir, "pkga", "pkga.go")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read fixture file: %v", err)
	}
	mutated := string(original) + "\n// ReconnectSentinel proves the offline mutation was reconciled.\nfunc ReconnectSentinel() {}\n"
	if err := os.WriteFile(target, []byte(mutated), 0o644); err != nil {
		t.Fatalf("write mutated fixture file: %v", err)
	}

	// The exact reconcile serve.go's RunE runs before mcp.BuildServer.
	stats, err := indexer.Sync(dir, storeDir, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("reconcile Sync: %v", err)
	}
	if stats.FilesReparsed == 0 {
		t.Fatal("reconcile Sync: got 0 files reparsed, want the offline-mutated file reparsed")
	}

	s := BuildServer(true, map[string]bool{}, dir, dir)

	result := callTool(t, s, "codegraph_explore", map[string]any{
		"query": "ReconnectSentinel",
		"path":  dir,
	})
	if result.IsError {
		t.Fatalf("codegraph_explore returned an error result: %+v", result)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "ReconnectSentinel") {
		t.Fatalf("codegraph_explore after reconcile: got %q, want it to find the offline-added ReconnectSentinel symbol", text)
	}

	// A second reconcile with nothing changed must be a cheap no-op.
	noopStats, err := indexer.Sync(dir, storeDir, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("no-op reconcile Sync: %v", err)
	}
	if noopStats.FilesReparsed != 0 {
		t.Fatalf("no-op reconcile Sync: got %d files reparsed, want 0 (nothing changed)", noopStats.FilesReparsed)
	}
}
