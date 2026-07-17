package present

import (
	"bytes"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
)

func fixtureFileTree() []*query.FileTreeNode {
	return []*query.FileTreeNode{
		{
			Name:  "internal",
			IsDir: true,
			Children: []*query.FileTreeNode{
				{Name: "main.go", Path: "internal/main.go", Language: "go"},
			},
		},
		{Name: "README.md", Path: "README.md", Language: "markdown"},
	}
}

func fixtureFileEntries() []query.FileEntry {
	return []query.FileEntry{
		{Path: "README.md", Language: "markdown"},
		{Path: "internal/main.go", Language: "go"},
	}
}

// TestRenderFiles_Tree covers the tree format: ANSI presence plus, ANSI-
// stripped, every directory name (with trailing slash) and every leaf
// "Name (Language)" at the same nesting the plain printFileTree would
// emit.
func TestRenderFiles_Tree(t *testing.T) {
	r := query.FilesResult{Format: "tree", Tree: fixtureFileTree()}
	var buf bytes.Buffer
	if err := RenderFiles(r, &buf); err != nil {
		t.Fatalf("RenderFiles: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\x1b[")) {
		t.Errorf("expected an ANSI escape sequence in output, got:\n%s", buf.String())
	}

	stripped := stripANSI(buf.String())
	for _, want := range []string{"internal/", "main.go (go)", "README.md (markdown)"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("expected %q in ANSI-stripped output:\n%s", want, stripped)
		}
	}

	// Nesting: the leaf under internal/ must be indented relative to the
	// top-level README.md line.
	lines := strings.Split(strings.TrimRight(stripped, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), stripped)
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Errorf("expected leaf line %q to be indented under its directory", lines[1])
	}
}

// TestRenderFiles_Flat covers the flat format: ANSI presence plus, ANSI-
// stripped, every "Path (Language)" line.
func TestRenderFiles_Flat(t *testing.T) {
	r := query.FilesResult{Format: "flat", Files: fixtureFileEntries()}
	var buf bytes.Buffer
	if err := RenderFiles(r, &buf); err != nil {
		t.Fatalf("RenderFiles: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("\x1b[")) {
		t.Errorf("expected an ANSI escape sequence in output, got:\n%s", buf.String())
	}

	stripped := stripANSI(buf.String())
	for _, want := range []string{"README.md (markdown)", "internal/main.go (go)"} {
		if !strings.Contains(stripped, want) {
			t.Errorf("expected %q in ANSI-stripped output:\n%s", want, stripped)
		}
	}
}

// TestRenderFiles_Empty covers an empty FilesResult: no leaf lines, no
// panic, nil error.
func TestRenderFiles_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderFiles(query.FilesResult{Format: "flat"}, &buf); err != nil {
		t.Fatalf("RenderFiles (empty flat): %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty flat output, got:\n%s", buf.String())
	}

	buf.Reset()
	if err := RenderFiles(query.FilesResult{Format: "tree"}, &buf); err != nil {
		t.Fatalf("RenderFiles (empty tree): %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty tree output, got:\n%s", buf.String())
	}
}
