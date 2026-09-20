package present

import (
	"io"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// This file's seven renderers style the plain formats already printed by
// internal/cli's search/callers/callees/impact/affected RunE bodies
// (D-08, CLI-01). Every renderer consumes its plain struct read-only —
// counts, order and wording are never recomputed here (D-02); the styled
// output strips back byte-for-byte to what the plain path already prints
// (D-00's content contract, pinned by results_test.go). present stays
// env-blind: callers build Palette via NewPalette(mode.Dark) at the RunE
// boundary and this file never reads a TTY/env value.

// writeLocationLine writes one "Name (Kind) FilePath:Line\n" row, matching
// the plain "%s (%s) %s:%d\n" format exactly: the unstyled separators
// (" (", ") ", ":", "\n") are what make the ANSI-stripped bytes equal the
// plain bytes. name/kind/filePath are repo-derived and may be adversarial
// — each passes through sanitizeControl before styling (CR-01).
func writeLocationLine(b *strings.Builder, pal Palette, indent, name, kind, filePath string, line int32) {
	b.WriteString(indent)
	b.WriteString(pal.Value.Render(sanitizeControl(name)))
	b.WriteString(" (")
	b.WriteString(pal.Label.Render(sanitizeControl(kind)))
	b.WriteString(") ")
	b.WriteString(pal.Path.Render(sanitizeControl(filePath) + ":" + strconv.Itoa(int(line))))
	b.WriteString("\n")
}

// writeNodeLine writes writeLocationLine's row for a *schema.Node — the
// shared entry point RenderSearchFull uses for a node's line 1 (identical
// to the plain default line).
func writeNodeLine(b *strings.Builder, pal Palette, n *schema.Node) {
	writeLocationLine(b, pal, "", n.Name, n.Kind, n.FilePath, n.StartLine)
}

// RenderSearch renders locs — the search default shape (D-01), one
// writeLocationLine per entry, no header, no indent. Mirrors search.go's
// plain loop over query.Location exactly.
func RenderSearch(locs []query.Location, pal Palette, w io.Writer) error {
	var b strings.Builder
	for _, l := range locs {
		writeLocationLine(&b, pal, "", l.Name, l.Kind, l.FilePath, l.StartLine)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderSearchFull renders nodes — the search --full shape (D-01): per
// node, line 1 via writeNodeLine, then line 2 ("    " + QualifiedName,
// plus "  " + Signature when non-empty) — the exact concatenation
// search.go's renderFullLine builds.
func RenderSearchFull(nodes []*schema.Node, pal Palette, w io.Writer) error {
	var b strings.Builder
	for _, n := range nodes {
		writeNodeLine(&b, pal, n)
		b.WriteString("    ")
		b.WriteString(pal.Value.Render(sanitizeControl(n.QualifiedName)))
		if n.Signature != "" {
			b.WriteString("  ")
			b.WriteString(pal.Label.Render(sanitizeControl(n.Signature)))
		}
		b.WriteString("\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderCallers renders r — the callers shape: header "Symbol has N
// caller(s):\n" (Symbol → Value, N → Count), then each entry indented
// "  " via writeLocationLine. Mirrors callers.go's plain branch exactly.
func RenderCallers(r query.CallersResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	b.WriteString(pal.Value.Render(sanitizeControl(r.Symbol)))
	b.WriteString(" has ")
	b.WriteString(pal.Count.Render(strconv.Itoa(len(r.Callers))))
	b.WriteString(" caller(s):\n")
	for _, l := range r.Callers {
		writeLocationLine(&b, pal, "  ", l.Name, l.Kind, l.FilePath, l.StartLine)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderCallees renders r — the callees shape: header "Symbol calls N
// callee(s):\n", then each entry indented "  ". Mirrors callees.go's
// plain branch exactly.
func RenderCallees(r query.CalleesResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	b.WriteString(pal.Value.Render(sanitizeControl(r.Symbol)))
	b.WriteString(" calls ")
	b.WriteString(pal.Count.Render(strconv.Itoa(len(r.Callees))))
	b.WriteString(" callee(s):\n")
	for _, l := range r.Callees {
		writeLocationLine(&b, pal, "  ", l.Name, l.Kind, l.FilePath, l.StartLine)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderImpact renders r — the impact shape: header "Symbol impact
// (depth=D): N node(s), M edge(s)\n" with D/N/M each styled Count, then
// each entry indented "  ". Mirrors impact.go's plain branch exactly.
func RenderImpact(r query.ImpactResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	b.WriteString(pal.Value.Render(sanitizeControl(r.Symbol)))
	b.WriteString(" impact (depth=")
	b.WriteString(pal.Count.Render(strconv.Itoa(r.Depth)))
	b.WriteString("): ")
	b.WriteString(pal.Count.Render(strconv.Itoa(r.NodeCount)))
	b.WriteString(" node(s), ")
	b.WriteString(pal.Count.Render(strconv.Itoa(r.EdgeCount)))
	b.WriteString(" edge(s)\n")
	for _, l := range r.Affected {
		writeLocationLine(&b, pal, "  ", l.Name, l.Kind, l.FilePath, l.StartLine)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderAffected renders r — the affected shape: zero entries →
// "no test files affected\n" (Label-styled); else "N affected
// test(s):\n" (N → Count) then each entry indented "  ". Mirrors
// affected.go's human (post-quiet) plain branch exactly.
func RenderAffected(r query.AffectedResult, pal Palette, w io.Writer) error {
	var b strings.Builder
	if len(r.AffectedTests) == 0 {
		b.WriteString(pal.Label.Render("no test files affected"))
		b.WriteString("\n")
		_, err := io.WriteString(w, b.String())
		return err
	}
	b.WriteString(pal.Count.Render(strconv.Itoa(len(r.AffectedTests))))
	b.WriteString(" affected test(s):\n")
	for _, l := range r.AffectedTests {
		writeLocationLine(&b, pal, "  ", l.Name, l.Kind, l.FilePath, l.StartLine)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// RenderNotice renders the WORK-02 worktree notice: empty writes nothing;
// non-empty is styled Warning line-by-line (preserving blank lines and
// the trailing newline structure exactly) so the stripped bytes equal
// query.WorktreeNotice's plain string verbatim.
func RenderNotice(notice string, pal Palette, w io.Writer) error {
	if notice == "" {
		return nil
	}
	trailingNL := strings.HasSuffix(notice, "\n")
	body := notice
	if trailingNL {
		body = notice[:len(notice)-1]
	}
	lines := strings.Split(body, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if line == "" {
			continue
		}
		b.WriteString(pal.Warning.Render(sanitizeControl(line)))
	}
	if trailingNL {
		b.WriteString("\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
}
