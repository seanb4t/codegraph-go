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
// headerStyle/labelStyle applied as structural chrome only. Never
// recomputes or re-sorts the tree (D-02) — nodes arrive already built and
// sorted by query.Engine.Files.
func writeFileTree(b *strings.Builder, nodes []*query.FileTreeNode, indent string) {
	for _, n := range nodes {
		if n.IsDir {
			fmt.Fprintf(b, "%s%s\n", indent, headerStyle.Render(n.Name+"/"))
			writeFileTree(b, n.Children, indent+"  ")
		} else {
			fmt.Fprintf(b, "%s%s (%s)\n", indent, n.Name, labelStyle.Render(n.Language))
		}
	}
}

// RenderFiles writes a lipgloss-styled rendering of r to w, mirroring the
// plain branch selection in internal/cli/files.go's RunE: r.Format ==
// "tree" walks the FileTreeNode slice via writeFileTree; otherwise each
// FileEntry is rendered as a styled "Path (Language)" line. r is consumed
// read-only — the tree structure and file ordering are never recomputed
// here (D-02). Callers gate this behind ChoosePresentation (D-03);
// RenderFiles itself never reads a TTY/env value.
func RenderFiles(r query.FilesResult, w io.Writer) error {
	var b strings.Builder
	if r.Format == "tree" {
		writeFileTree(&b, r.Tree, "")
	} else {
		for _, f := range r.Files {
			fmt.Fprintf(&b, "%s (%s)\n", f.Path, labelStyle.Render(f.Language))
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}
