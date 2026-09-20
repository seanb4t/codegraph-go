package present

import (
	"io"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// pluralize mirrors internal/query/render_markdown.go's unexported
// pluralize (D-07: the styled wording must match the markdown's exactly)
// — duplicated here per the present/status.go formatNumber precedent,
// since internal/query stays frozen this phase.
func pluralize(n int, s string) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

// sourceDisclaimerText duplicates internal/query/render_markdown.go's
// unexported sourceDisclaimer, with its embedded "**...**" markdown bold
// markers already removed — the styled renderer never emits markdown
// syntax (D-07), so this constant is what stripANSI(styled) must equal
// once markdownToPlainContract has stripped the plain sibling's "> "
// blockquote prefix and its own embedded "**...**" markers.
const sourceDisclaimerText = "The code below is the verbatim, current on-disk source of these files — re-read from disk on this call and line-numbered, byte-for-byte identical to what the Read tool returns. It is NOT a summary, outline, or stale cache. Treat each block as a Read you have already performed: do not Read a file shown here."

// RenderExplore writes a lipgloss-styled rendering of r — the SAME
// sections, order and wording as query.RenderExplore's (and, for the
// empty case, exploreZeroResult's) markdown (D-07): hue replaces markdown
// syntax (**, backticks, "> ", fences), nothing is re-laid-out. r is
// consumed read-only; every repo/user-derived string passes
// sanitizeControl before styling (CR-01). present reads no file — source
// bytes come only from r.Sources (T-04-16).
func RenderExplore(r query.ExploreResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	if r.Stale {
		b.WriteString(pal.Warning.Render("⚠ Index may be stale — a sync is pending.") + "\n\n")
	}
	b.WriteString(pal.Header.Render("Exploration: "+sanitizeControl(r.Query)) + "\n\n")

	if r.Empty {
		b.WriteString("Found " + pal.Count.Render("0") + " symbols across " + pal.Count.Render("0") + " files.\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	fileCount := len(r.Groups)
	b.WriteString("Found " + pal.Count.Render(strconv.Itoa(r.SymbolCount)) + " " + pluralize(r.SymbolCount, "symbol") +
		" across " + pal.Count.Render(strconv.Itoa(fileCount)) + " " + pluralize(fileCount, "file") + ".\n\n")

	b.WriteString(pal.Header.Render("Blast radius — what depends on these (update/verify before editing)") + "\n\n")
	for _, bl := range r.Blasts {
		writeBlastBullet(&b, pal, bl)
		b.WriteString("\n")
	}

	b.WriteString("\n" + pal.Header.Render("Source Code") + "\n\n")
	b.WriteString(pal.Label.Render(sourceDisclaimerText) + "\n\n")

	for _, g := range r.Groups {
		b.WriteString(pal.Path.Render(sanitizeControl(g.Path)) + " — " + pal.Label.Render(joinSymbolKindList(g.Symbols)) + "\n\n")
		if r.SkeletonFiles[g.Path] {
			writeSkeleton(&b, pal, g.Symbols)
		} else {
			writeNumberedSource(&b, pal, r.Sources[g.Path])
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// writeBlastBullet writes one blast-radius bullet, mirroring
// internal/query/render_markdown.go's unexported renderBlastBullet
// exactly minus markdown syntax (D-07): symbol name, its own file:line,
// caller count, then EITHER the covering-tests list OR the "no covering
// tests found" warning (CallerCount>0 only) OR nothing (CallerCount==0).
func writeBlastBullet(b *strings.Builder, pal Palette, bl query.ExploreBlast) {
	n := bl.Symbol
	b.WriteString("- ")
	b.WriteString(pal.Value.Render(sanitizeControl(n.Name)))
	b.WriteString(" (")
	b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath) + ":" + strconv.Itoa(int(n.StartLine))))
	b.WriteString(") — ")
	b.WriteString(pal.Count.Render(strconv.Itoa(bl.CallerCount)))
	b.WriteString(" " + pluralize(bl.CallerCount, "caller") + " in ")
	b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath)))
	switch {
	case len(bl.TestFiles) > 0:
		parts := make([]string, len(bl.TestFiles))
		for i, f := range bl.TestFiles {
			parts[i] = pal.Path.Render(sanitizeControl(f))
		}
		b.WriteString("; tests: " + strings.Join(parts, ", "))
	case bl.CallerCount > 0:
		b.WriteString("; " + pal.Warning.Render("⚠️ no covering tests found"))
	}
}

// joinSymbolKindList renders explore's per-file header symbol list —
// mirrors render_markdown.go's unexported joinSymbolKindList minus
// styling: "name1(kind1), name2(kind2)".
func joinSymbolKindList(nodes []*schema.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = sanitizeControl(n.Name) + "(" + sanitizeControl(n.Kind) + ")"
	}
	return strings.Join(parts, ", ")
}

// writeNumberedSource writes content's significant lines as
// "<n>\t<line>\n" rows (Label for the line number, Value for the line
// itself) — the fence-free equivalent of render_markdown.go's unexported
// renderNumberedSource (its "```go\n"/"```\n" fence lines are markdown
// syntax the contract removes, D-07); a trailing empty final element from
// an on-disk trailing newline is dropped, exactly as the plain renderer
// does.
func writeNumberedSource(b *strings.Builder, pal Palette, content []byte) {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		b.WriteString(pal.Label.Render(strconv.Itoa(i + 1)))
		b.WriteString("\t")
		b.WriteString(pal.Value.Render(sanitizeControl(line)))
		b.WriteString("\n")
	}
}

// writeNumberedSourceRange is writeNumberedSource's sibling for a
// definition's own [startLine,endLine] span — mirrors render_markdown.go's
// unexported renderNumberedSourceRange's clamping exactly (true on-disk
// line numbers, not renumbered from 1; an out-of-range end clamps to the
// last line, an out-of-range start clamps into range).
func writeNumberedSourceRange(b *strings.Builder, pal Palette, content []byte, startLine, endLine int32) {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	start, end := int(startLine), int(endLine)
	if start < 1 {
		start = 1
	}
	if start > len(lines) {
		start = len(lines)
	}
	if end < start || end > len(lines) {
		end = len(lines)
	}
	for i := start; i <= end && i >= 1; i++ {
		b.WriteString(pal.Label.Render(strconv.Itoa(i)))
		b.WriteString("\t")
		b.WriteString(pal.Value.Render(sanitizeControl(lines[i-1])))
		b.WriteString("\n")
	}
}

// writeSkeleton writes H20's signature-only view: one Value-styled
// "<kind> <name><signature>" line per symbol — the fence-free equivalent
// of render_markdown.go's unexported renderSkeleton.
func writeSkeleton(b *strings.Builder, pal Palette, symbols []*schema.Node) {
	for _, n := range symbols {
		b.WriteString(pal.Value.Render(sanitizeControl(n.Kind) + " " + sanitizeControl(n.Name) + sanitizeControl(n.Signature)))
		b.WriteString("\n")
	}
}
