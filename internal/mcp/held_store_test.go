package mcp

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore/locktest"
)

// TestToolCallOnHeldStore checks the result of a tool call on a store that
// another holder keeps past the retry, and the result of the same call
// after the holder releases the store.
func TestToolCallOnHeldStore(t *testing.T) {
	dir := copyFixture(t)
	indexFixture(t, dir)
	s := BuildServer(true, map[string]bool{"explore": true}, dir, dir)
	args := map[string]any{"query": "Alpha"}

	release := locktest.HoldStoreLock(t, filepath.Join(dir, ".codegraph", "store"))
	held := callTool(t, s, "codegraph_explore", args)
	if !held.IsError {
		t.Fatalf("codegraph_explore on a held store returned no error result: %q", resultText(t, held))
	}
	if text := resultText(t, held); !strings.Contains(text, "store lock held") {
		t.Fatalf("the error result does not say that the store lock is held: %q", text)
	}

	release()
	free := callTool(t, s, "codegraph_explore", args)
	if free.IsError {
		t.Fatalf("codegraph_explore after the release returned an error result: %q", resultText(t, free))
	}
	if text := resultText(t, free); !strings.Contains(text, "Alpha") {
		t.Fatalf("the result after the release does not hold the symbol Alpha: %q", text)
	}
}
