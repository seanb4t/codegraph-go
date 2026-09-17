package present

import (
	"fmt"
	"io"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// writeFileTree writes a styled rendering of a query.FileTreeNode slice —
// the same shape internal/cli/files.go's printFileTree walks (directory
// nodes get a trailing slash, leaf nodes get "Name (Language)"), with
// pal.Path/pal.Label applied as structural chrome only. Never recomputes
// or re-sorts the tree (D-02) — nodes arrive already built and sorted by
// query.Engine.Files.
func writeFileTree(b *strings.Builder, pal Palette, nodes []*query.FileTreeNode, indent string) {
	for _, n := range nodes {
		if n.IsDir {
			// n.Name is filesystem-derived and may be adversarial — strip
			// control characters before it reaches the terminal (CR-01).
			fmt.Fprintf(b, "%s%s\n", indent, pal.Path.Render(sanitizeControl(n.Name)+"/"))
			writeFileTree(b, pal, n.Children, indent+"  ")
		} else {
			fmt.Fprintf(b, "%s%s (%s)\n", indent, pal.Path.Render(sanitizeControl(n.Name)), pal.Label.Render(n.Language))
		}
	}
}

// RenderFiles writes a lipgloss-styled rendering of r to w, mirroring the
// plain branch selection in internal/cli/files.go's RunE: r.Format ==
// "tree" walks the FileTreeNode slice via writeFileTree; otherwise each
// FileEntry is rendered as a styled "Path (Language)" line. r is consumed
// read-only — the tree structure and file ordering are never recomputed
// here (D-02). Callers gate this behind ChoosePresentation (D-03) and
// build pal via NewPalette(mode.Dark) at the RunE boundary; RenderFiles
// itself never reads a TTY/env value.
func RenderFiles(r query.FilesResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	if r.Format == "tree" {
		writeFileTree(&b, pal, r.Tree, "")
	} else {
		for _, f := range r.Files {
			// f.Path is filesystem-derived and may be adversarial — strip
			// control characters before it reaches the terminal (CR-01).
			fmt.Fprintf(&b, "%s (%s)\n", pal.Path.Render(sanitizeControl(f.Path)), pal.Label.Render(f.Language))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
