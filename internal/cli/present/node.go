package present

import (
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// nodeMultiDefHardCap and nodeMultiDefBodyBudget duplicate internal/query/render_markdown.go's unexported NODE-02 budget
// constants verbatim (present/status.go's formatNumber precedent) —
// internal/query must not export or change them this phase (D-07). Kept
// in lockstep with render_markdown.go by review.
const (
	nodeMultiDefHardCap    = 16
	nodeMultiDefBodyBudget = 12000
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
// unconditional lines). A data kind adds Accessed by ← and
// Initialized to →, also when empty.
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
	b.WriteString(pal.Header.Render(query.TrailHeader) + "\n")
	for _, row := range trailRows(pal, d) {
		b.WriteString(pal.Label.Render(row.label) + " " + row.refs() + "\n")
	}
}

// trailRow is one styled trail line: its label, its number of entries and
// a function that writes the entries.
type trailRow struct {
	label string
	n     int
	refs  func() string
}

// trailRows returns the styled trail lines of d in the order of the
// markdown view: calls and callers, then the data lines for a variable, a
// constant or a field.
func trailRows(pal Palette, d *query.DefinitionDetail) []trailRow {
	rows := []trailRow{
		{"Calls →", len(d.Calls), func() string { return writeNodeRefs(pal, d.Calls, nil) }},
		{"Called by ←", len(d.CalledBy), func() string { return writeNodeRefs(pal, d.CalledBy, nil) }},
	}
	if goextract.IsDataKind(d.Node.Kind) {
		rows = append(rows,
			trailRow{"Accessed by ←", len(d.AccessedBy), func() string { return writeAccessRefs(pal, d.AccessedBy) }},
			trailRow{"Initialized to →", len(d.InitializedTo), func() string { return writeNodeRefs(pal, d.InitializedTo, nil) }},
		)
	}
	return rows
}

// writeNodeRefs comma-joins "name (path:line)" references, mirroring
// render_markdown.go's unexported joinNodeRefs/formatNodeRef minus
// markdown syntax — preserves input order (Calls → forward, Called by ←
// reverse-adjacency order), exactly as the plain renderer does. A non-nil
// suffix gives the suffix of entry i, which goes inside the parentheses.
func writeNodeRefs(pal Palette, nodes []*schema.Node, suffix func(i int) string) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		tail := ""
		if suffix != nil {
			if s := suffix(i); s != "" {
				tail = ", " + pal.Value.Render(sanitizeControl(s))
			}
		}
		parts[i] = pal.Value.Render(sanitizeControl(n.Name)) + " (" + pal.Path.Render(sanitizeControl(n.FilePath)+":"+strconv.Itoa(int(n.StartLine))) + tail + ")"
	}
	return strings.Join(parts, ", ")
}

// writeAccessRefs writes access entries as "name (path:line, role)".
func writeAccessRefs(pal Palette, access []query.DataAccess) string {
	nodes := make([]*schema.Node, len(access))
	for i, a := range access {
		nodes[i] = a.Node
	}
	return writeNodeRefs(pal, nodes, func(i int) string { return access[i].Access })
}

// writeNodeSection renders one multi-def candidate's full detail as a
// single string — mirrors render_markdown.go's unexported
// renderNodeSection minus markdown syntax exactly, including its Trail
// block being OMITTED ENTIRELY when both calls and calledBy are empty
// (unlike writeSingleDef, which always renders it): name/kind, blank,
// Location, Signature, blank, the numbered [StartLine,EndLine] range
// (trailing newline trimmed, matching the plain "parts" join having no
// trailing newline of its own). The data lines print only when their list
// has an entry, and any non-empty list of the four prints the Trail line.
func writeNodeSection(pal Palette, d *query.DefinitionDetail) string {
	n := d.Node
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
	writeNumberedSourceRange(&rangeBuilder, pal, d.Source, n.StartLine, n.EndLine)
	b.WriteString(strings.TrimSuffix(rangeBuilder.String(), "\n"))

	header := false
	for _, row := range trailRows(pal, d) {
		if row.n == 0 {
			continue
		}
		if !header {
			b.WriteString("\n")
			b.WriteString(pal.Header.Render(query.TrailHeader))
			header = true
		}
		b.WriteString("\n")
		b.WriteString(pal.Label.Render(row.label) + " " + row.refs())
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
		section := writeNodeSection(pal, dd)
		if len(rendered) == 0 || used+strippedLen(section) <= nodeMultiDefBodyBudget {
			rendered = append(rendered, section)
			used += strippedLen(section)
		} else {
			listed = append(listed, n)
		}
	}

	b.WriteString(pal.Header.Render(query.DefinitionsHeading(len(m.Matches), m.Symbol)))
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
		capped, more := query.CapDefinitions(listed)
		for _, n := range capped {
			b.WriteString("- ")
			b.WriteString(pal.Value.Render(sanitizeControl(n.Name)))
			b.WriteString(" (")
			b.WriteString(pal.Label.Render(sanitizeControl(n.Kind)))
			b.WriteString(") — ")
			b.WriteString(pal.Path.Render(sanitizeControl(n.FilePath) + ":" + strconv.Itoa(int(n.StartLine))))
			b.WriteString("\n")
		}
		if more > 0 {
			b.WriteString("- … +" + strconv.Itoa(more) + " more\n")
		}
		b.WriteString("\n")
		hint := "Need one of these in full? Call codegraph_node again with file (e.g. " + strconv.Quote(path.Base(listed[0].FilePath)) + ") or line — do NOT Read it."
		b.WriteString(pal.Label.Render(hint))
		b.WriteString("\n")
	}
	return nil
}
