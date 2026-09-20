package present

import (
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// nodeMultiDefHardCap, nodeMultiDefBodyBudget and nodeMultiDefListCap
// duplicate internal/query/render_markdown.go's unexported NODE-02 budget
// constants verbatim (present/status.go's formatNumber precedent) —
// internal/query must not export or change them this phase (D-07). Kept
// in lockstep with render_markdown.go by review.
const (
	nodeMultiDefHardCap    = 16
	nodeMultiDefBodyBudget = 12000
	nodeMultiDefListCap    = 20
)

// nodeSectionAnsi matches an SGR escape sequence — used ONLY to measure a
// multi-def section's byte length for the BODY_BUDGET decision (T-04-17),
// so the decision matches the plain renderer's own budget arithmetic
// exactly. A local pattern (not charmbracelet/x/ansi, D-15), distinct
// from ansistrip_test.go's test-only stripANSI, which production code in
// this file cannot import.
var nodeSectionAnsi = regexp.MustCompile("\x1b\\[[0-9;]*m")

// strippedLen returns s's length with every SGR escape sequence removed —
// the byte count a plain-text budget decision needs, computed without
// duplicating the whole section-rendering algorithm a second time.
func strippedLen(s string) int {
	return len(nodeSectionAnsi.ReplaceAllString(s, ""))
}

// RenderNode writes a lipgloss-styled rendering of d — the SAME sections,
// order and wording as query.RenderNode/RenderNodeMultiDef's markdown
// (D-07). Every mode consumes its detail struct read-only; present reads
// no file (T-04-16) — File mode's source is d.File.Source, SingleDef's
// calls/calledBy are exactly what the engine fetched, MultiDef fetches
// each candidate lazily via d.Multi.Definition(n), stopping at HARD_CAP.
func RenderNode(d query.NodeDetail, pal Palette, w io.Writer) error {
	var b strings.Builder
	switch d.Mode {
	case query.NodeDetailModeFile:
		writeNumberedSource(&b, pal, d.File.Source)
	case query.NodeDetailModeSingleDef:
		writeSingleDef(&b, pal, d.Definition)
	default:
		if err := writeMultiDef(&b, pal, d.Multi); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// writeSingleDef mirrors internal/query/render_markdown.go's exported
// RenderNode minus markdown syntax (D-07): name/kind, blank, Location,
// Signature, the fixed Trail line, Calls →, Called by ← — always
// rendered, even when a list is empty (matching RenderNode's own
// unconditional lines).
func writeSingleDef(b *strings.Builder, pal Palette, d *query.DefinitionDetail) {
	n := d.Node
	b.WriteString(pal.Value.Render(sanitizeControl(n.Name)))
	b.WriteString(" (")
	b.WriteString(pal.Label.Render(sanitizeControl(n.Kind)))
	b.WriteString(")\n\n")
	b.WriteString(pal.Label.Render("Location:") + " ")
	b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath) + ":" + strconv.Itoa(int(n.StartLine))))
	b.WriteString("\n")
	b.WriteString(pal.Label.Render("Signature:") + " ")
	b.WriteString(pal.Value.Render(sanitizeControl(n.Signature)))
	b.WriteString("\n")
	b.WriteString(pal.Header.Render("Trail — codegraph_node any of these to follow it (no Read needed)") + "\n")
	b.WriteString(pal.Label.Render("Calls →") + " " + writeNodeRefs(pal, d.Calls) + "\n")
	b.WriteString(pal.Label.Render("Called by ←") + " " + writeNodeRefs(pal, d.CalledBy) + "\n")
}

// writeNodeRefs comma-joins "name (path:line)" references, mirroring
// render_markdown.go's unexported joinNodeRefs/formatNodeRef minus
// markdown syntax — preserves input order (Calls → forward, Called by ←
// reverse-adjacency order), exactly as the plain renderer does.
func writeNodeRefs(pal Palette, nodes []*schema.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = pal.Value.Render(sanitizeControl(n.Name)) + " (" + pal.Path.Render(sanitizeControl(n.FilePath)+":"+strconv.Itoa(int(n.StartLine))) + ")"
	}
	return strings.Join(parts, ", ")
}

// writeNodeSection renders one multi-def candidate's full detail as a
// single string — mirrors render_markdown.go's unexported
// renderNodeSection minus markdown syntax exactly, including its Trail
// block being OMITTED ENTIRELY when both calls and calledBy are empty
// (unlike writeSingleDef, which always renders it): name/kind, blank,
// Location, Signature, blank, the numbered [StartLine,EndLine] range
// (trailing newline trimmed, matching the plain "parts" join having no
// trailing newline of its own).
func writeNodeSection(pal Palette, n *schema.Node, source []byte, calls, calledBy []*schema.Node) string {
	var b strings.Builder
	b.WriteString(pal.Value.Render(sanitizeControl(n.Name)))
	b.WriteString(" (")
	b.WriteString(pal.Label.Render(sanitizeControl(n.Kind)))
	b.WriteString(")\n\n")
	b.WriteString(pal.Label.Render("Location:") + " ")
	b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath) + ":" + strconv.Itoa(int(n.StartLine))))
	b.WriteString("\n")
	b.WriteString(pal.Label.Render("Signature:") + " ")
	b.WriteString(pal.Value.Render(sanitizeControl(n.Signature)))
	b.WriteString("\n\n")

	var rangeBuilder strings.Builder
	writeNumberedSourceRange(&rangeBuilder, pal, source, n.StartLine, n.EndLine)
	b.WriteString(strings.TrimSuffix(rangeBuilder.String(), "\n"))

	if len(calls) > 0 || len(calledBy) > 0 {
		b.WriteString("\n")
		b.WriteString(pal.Header.Render("Trail — codegraph_node any of these to follow it (no Read needed)"))
		if len(calls) > 0 {
			b.WriteString("\n")
			b.WriteString(pal.Label.Render("Calls →") + " " + writeNodeRefs(pal, calls))
		}
		if len(calledBy) > 0 {
			b.WriteString("\n")
			b.WriteString(pal.Label.Render("Called by ←") + " " + writeNodeRefs(pal, calledBy))
		}
	}
	return b.String()
}

// writeMultiDef reproduces RenderNodeMultiDef's exact loop (D-07,
// T-04-17): fetches each candidate LAZILY via m.Definition(n), in match
// order, stopping fetches once HARD_CAP renders have been produced;
// budgets the cumulative rendered length against BODY_BUDGET (the first
// section always renders regardless); lists overflow definitions capped
// at LIST_CAP with a trailing "+K more", plus the closing hint line.
func writeMultiDef(b *strings.Builder, pal Palette, m *query.MultiDefDetail) error {
	var rendered []string
	var listed []*schema.Node
	used := 0
	for _, n := range m.Matches {
		if len(rendered) >= nodeMultiDefHardCap {
			listed = append(listed, n)
			continue
		}
		dd, err := m.Definition(n)
		if err != nil {
			return err
		}
		section := writeNodeSection(pal, n, dd.Source, dd.Calls, dd.CalledBy)
		if len(rendered) == 0 || used+strippedLen(section) <= nodeMultiDefBodyBudget {
			rendered = append(rendered, section)
			used += strippedLen(section)
		} else {
			listed = append(listed, n)
		}
	}

	b.WriteString(pal.Header.Render(strconv.Itoa(len(m.Matches)) + " definitions named " + strconv.Quote(m.Symbol)))
	b.WriteString("\n")
	overflow := ""
	if len(listed) > 0 {
		overflow = "; " + strconv.Itoa(len(listed)) + " more listed below"
	}
	b.WriteString("Returning " + pal.Count.Render(strconv.Itoa(len(rendered))) + " in full" + overflow + " — pick the one you need (no Read required).\n\n")
	b.WriteString(strings.Join(rendered, "\n\n---\n\n"))
	b.WriteString("\n")

	if len(listed) > 0 {
		b.WriteString("\n")
		b.WriteString(pal.Header.Render("Other definitions"))
		b.WriteString("\n")
		capped := listed
		if len(capped) > nodeMultiDefListCap {
			capped = capped[:nodeMultiDefListCap]
		}
		for _, n := range capped {
			b.WriteString("- ")
			b.WriteString(pal.Value.Render(sanitizeControl(n.Name)))
			b.WriteString(" (")
			b.WriteString(pal.Label.Render(sanitizeControl(n.Kind)))
			b.WriteString(") — ")
			b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath) + ":" + strconv.Itoa(int(n.StartLine))))
			b.WriteString("\n")
		}
		if len(listed) > nodeMultiDefListCap {
			b.WriteString("- … +" + strconv.Itoa(len(listed)-nodeMultiDefListCap) + " more\n")
		}
		b.WriteString("\n")
		hint := "Need one of these in full? Call codegraph_node again with file (e.g. " + strconv.Quote(path.Base(listed[0].FilePath)) + ") or line — do NOT Read it."
		b.WriteString(pal.Label.Render(hint))
		b.WriteString("\n")
	}
	return nil
}
