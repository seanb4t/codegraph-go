package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncCmdErrorsWhenUninitialized(t *testing.T) {
	freshDir := t.TempDir()
	_, _, err := execCmd("sync", freshDir)
	if !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestSyncCmdUpdatesGraph(t *testing.T) {
	dir := copyFixture(t)
	codegraphDir := filepath.Join(dir, codegraphDirName)

	if _, _, err := execCmd("init", dir); err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	nodesBefore, _ := readGraphCounts(t, codegraphDir)

	// Edit main.go so sync has a genuine change to reparse (content-hash
	// diff, D-01a) — a new top-level function adds a node.
	mainPath := filepath.Join(dir, "main.go")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	edited := string(data) + "\nfunc helper() {}\n"
	if err := os.WriteFile(mainPath, []byte(edited), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	out, _, err := execCmd("sync", dir)
	if err != nil {
		t.Fatalf("sync: unexpected error: %v", err)
	}
	if !strings.Contains(out, "files=") {
		t.Fatalf("expected summary output to mention files, got %q", out)
	}
	if !strings.Contains(out, "reparsed=") {
		t.Fatalf("expected sync summary to mention reparsed, got %q", out)
	}

	nodesAfter, _ := readGraphCounts(t, codegraphDir)
	if nodesAfter <= nodesBefore {
		t.Fatalf("expected node count to grow after adding a function: before=%d after=%d", nodesBefore, nodesAfter)
	}
}
