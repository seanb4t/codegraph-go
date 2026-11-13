// Package query (this file): the SURF-06 markdown renderers for the 5
// JSON-shaped MCP read tools (callers/callees/impact/search/files).
//
// This file is strictly ADDITIVE (D-16). Every one of the corresponding
// Marshal*JSON helpers (traverse.go, files.go) is SHARED with the CLI
// --json path AND is testdata/golden's shape oracle — e.g.
// MarshalCallersJSON is called from both internal/cli/callers.go and
// internal/mcp/tools.go. Mutating one of those bodies to emit markdown
// would silently break the CLI contract and the behavioral golden suite
// simultaneously. So after this phase each helper family has exactly one
// caller per surface: Marshal*JSON is called ONLY by internal/cli (the
// --json contract and behavioral_test.go's shape oracle); Render*
// (this file) is called ONLY by internal/mcp (whose consumer is a
// language model, not a parser — nothing unmarshals MCP text content).
// That asymmetry is intentional, not an oversight — do not "helpfully"
// reunify the two families.
package query

import (
	"fmt"
	"strings"
)

// renderLocationTable renders locs as a markdown table — a header row
// once, then one row per record, in INPUT ORDER. It backs all four
// Location-shaped renderers (RenderCallersMarkdown, RenderCalleesMarkdown,
// RenderImpactMarkdown, RenderSearchMarkdown), since Location{Name, Kind,
// FilePath, StartLine} is the same record type across callers/callees/
// impact/search.
//
// Every result slice reaching this function is ALREADY deterministically
// ordered upstream before it gets here: files sorts by path
// (files.go:149), search sorts by lexical tier (search.go:98), callers
// walks the sorted reverse adjacency (traverse.go:82), callees is a
// single contiguous storage-key-ordered range scan, and impact is a BFS
// over that same sorted adjacency. Do NOT add a sort call here — doing so
// would silently diverge MCP's row order from the CLI --json array order
// for the same query, reintroducing exactly the CLI/MCP drift the
// shared-engine rule exists to prevent.
//
// Returns "" for a zero-length slice; callers supply their own worded
// no-results sentence (each renderer names its own noun — "no callers",
// "no callees", etc.) rather than this shared helper guessing one.
func renderLocationTable(locs []Location) string {
	return renderLocationTableColumns(locs, false)
}

// renderLocationTableColumns is renderLocationTable with the two
// optional columns. The Overload column is present when one row carries
// a signature or is ambiguous. The Definition column is present when
// hasDefinition is set: the caller passes it from the result, never from
// the rows, so a query that spans definitions has the column also when
// every row is ambiguous and names none.
func renderLocationTableColumns(locs []Location, hasDefinition bool) string {
	if len(locs) == 0 {
		return ""
	}
	hasOverload := false
	for _, l := range locs {
		if l.Signature != "" || l.Ambiguous {
			hasOverload = true
			break
		}
	}
	cell := func(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
	var b strings.Builder
	b.WriteString("| Name | Kind | Location |")
	if hasDefinition {
		b.WriteString(" Definition |")
	}
	if hasOverload {
		b.WriteString(" Overload |")
	}
	b.WriteString("\n|---|---|---|")
	if hasDefinition {
		b.WriteString("---|")
	}
	if hasOverload {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, l := range locs {
		fmt.Fprintf(&b, "| `%s` | %s | `%s:%d` |", l.Name, l.Kind, l.FilePath, l.StartLine)
		if hasDefinition {
			if l.Definition != "" {
				fmt.Fprintf(&b, " `%s` |", cell(l.Definition))
			} else {
				b.WriteString("  |")
			}
		}
		if hasOverload {
			// A query over an overload set adds the overload each row
			// belongs to, or marks the row ambiguous.
			fmt.Fprintf(&b, " %s |", cell(OverloadLabel(l)))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// OverloadLabel is the text that marks a row of a query over overloads:
// the signature of the overload it belongs to, or "ambiguous". It is ""
// for a row of an ordinary query, so the plain and styled renderers
// print nothing extra there.
func OverloadLabel(l Location) string {
	switch {
	case l.Ambiguous:
		return "ambiguous"
	case l.Signature != "":
		return l.Signature
	}
	return ""
}

// OverloadSuffix is OverloadLabel in the form the plain location line
// appends: a space and the label in brackets, or "".
func OverloadSuffix(l Location) string {
	if label := OverloadLabel(l); label != "" {
		return " [" + label + "]"
	}
	return ""
}

// RowSuffix is the text the plain location line appends to a row: the
// overload suffix, then the definition the row belongs to when the query
// spans definitions. It is "" for a row of an ordinary query.
func RowSuffix(l Location) string {
	s := OverloadSuffix(l)
	if l.Definition != "" {
		s += " (definition " + l.Definition + ")"
	}
	return s
}

// DefinitionsHeading is the heading of a list of the definitions of one
// name. `node` and the definitions line of callers, callees and impact
// share it.
func DefinitionsHeading(n int, symbol string) string {
	return fmt.Sprintf("%d definitions named %q", n, symbol)
}

// capList returns the first max items and the count of the rest.
func capList[T any](items []T, max int) (shown []T, more int) {
	if len(items) > max {
		return items[:max], len(items) - max
	}
	return items, 0
}

// DefinitionsLine is the line that lists the definitions of a query that
// spans them: the heading, then the label of each definition, at most
// nodeMultiDefListCap of them. It is "" when defs is empty, which is the
// case for every query over one definition or one overload set.
func DefinitionsLine(symbol string, defs []Location) string {
	if len(defs) == 0 {
		return ""
	}
	shown, more := capList(defs, nodeMultiDefListCap)
	labels := make([]string, len(shown))
	for i, d := range shown {
		labels[i] = d.Definition
	}
	line := DefinitionsHeading(len(defs), symbol) + ": " + strings.Join(labels, ", ")
	if more > 0 {
		line += fmt.Sprintf(", +%d more", more)
	}
	return line
}

// definitionsParagraph is DefinitionsLine as a markdown paragraph after
// the header of a result, or "".
func definitionsParagraph(symbol string, defs []Location) string {
	if line := DefinitionsLine(symbol, defs); line != "" {
		return line + "\n\n"
	}
	return ""
}

// emptyDefinitions is the definitions line after the sentence of a
// result with no row, or "". A name with several definitions and no
// caller still says which definitions the query started from.
func emptyDefinitions(symbol string, defs []Location) string {
	if line := DefinitionsLine(symbol, defs); line != "" {
		return "\n" + line + "\n"
	}
	return ""
}

// RenderCallersMarkdown renders a CallersResult as markdown: a bolded
// header naming the symbol and caller count, then the shared location
// table. An empty result renders an explicit "no callers" sentence naming
// the symbol instead of a headerless table — a bare table header with
// zero rows reads as a rendering bug to a model.
func RenderCallersMarkdown(r CallersResult) string {
	if len(r.Callers) == 0 {
		return fmt.Sprintf("**Callers of `%s`** — no callers found.\n", r.Symbol) + emptyDefinitions(r.Symbol, r.Definitions)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Callers of `%s`** — %d %s\n\n", r.Symbol, len(r.Callers), pluralize(len(r.Callers), "caller"))
	b.WriteString(definitionsParagraph(r.Symbol, r.Definitions))
	b.WriteString(renderLocationTableColumns(r.Callers, len(r.Definitions) > 0))
	return b.String()
}

// RenderCalleesMarkdown renders a CalleesResult as markdown, mirroring
// RenderCallersMarkdown's shape for call targets instead of call sites.
func RenderCalleesMarkdown(r CalleesResult) string {
	if len(r.Callees) == 0 {
		return fmt.Sprintf("**Callees of `%s`** — no callees found.\n", r.Symbol) + emptyDefinitions(r.Symbol, r.Definitions)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Callees of `%s`** — %d %s\n\n", r.Symbol, len(r.Callees), pluralize(len(r.Callees), "callee"))
	b.WriteString(definitionsParagraph(r.Symbol, r.Definitions))
	b.WriteString(renderLocationTableColumns(r.Callees, len(r.Definitions) > 0))
	return b.String()
}

// RenderImpactMarkdown renders an ImpactResult as markdown. Unlike the
// other three Location-backed renderers, its header additionally carries
// Depth/NodeCount/EdgeCount — the scalars ImpactResult has that have no
// place in a per-row table — mirroring the documented bolded-key style.
func RenderImpactMarkdown(r ImpactResult) string {
	if len(r.Affected) == 0 {
		return fmt.Sprintf("**Impact of `%s`** — depth %d, %d nodes, %d edges — no affected symbols found.\n", r.Symbol, r.Depth, r.NodeCount, r.EdgeCount) + emptyDefinitions(r.Symbol, r.Definitions)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Impact of `%s`** — depth %d, %d nodes, %d edges\n\n", r.Symbol, r.Depth, r.NodeCount, r.EdgeCount)
	b.WriteString(definitionsParagraph(r.Symbol, r.Definitions))
	b.WriteString(renderLocationTableColumns(r.Affected, len(r.Definitions) > 0))
	return b.String()
}

// RenderSearchMarkdown renders search results as markdown. It takes the
// raw []Location slice (rather than a wrapper result struct) because
// Engine.Search returns []Location directly with no wrapper — the same
// reason search has no MarshalXJSON helper of its own.
func RenderSearchMarkdown(term string, locs []Location) string {
	if len(locs) == 0 {
		return fmt.Sprintf("**Search: `%s`** — no results found.\n", term)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Search: `%s`** — %d %s\n\n", term, len(locs), pluralize(len(locs), "result"))
	b.WriteString(renderLocationTable(locs))
	return b.String()
}

// RenderFilesMarkdown renders a FilesResult as markdown. FilesResult is a
// UNION — exactly one of Files (flat format) or Tree (tree format) is
// populated, per Format — so, unlike the four Location-backed renderers
// above, this one branches on shape rather than sharing a single table.
//
// Both the empty string and the literal "flat" value are treated as the
// flat branch, matching FilesOptions.Format's documented "empty means
// flat" default and the MCP files tool's own req.GetString("format", "")
// default.
//
// Note: the documented MCP files tool defaults to the tree format; ours defaults to
// flat. That is a PRE-EXISTING divergence, not introduced by this plan —
// do not "fix" it here; it is Phase 8 SURF territory.
func RenderFilesMarkdown(r FilesResult) string {
	if r.Format == "tree" {
		var b strings.Builder
		b.WriteString("**Files (tree)**\n\n")
		b.WriteString(renderFileTreeMarkdown(r.Tree, ""))
		return b.String()
	}

	// Flat branch (Format == "" or "flat"). A table cannot represent the
	// tree branch's nested shape, but the flat branch is a uniform record
	// list — this is where SURF-06's measured -41% win comes from: JSON
	// re-states Path/Language/Nodes/Edges on every one of 308 records; a
	// table states them once.
	if len(r.Files) == 0 {
		return "**Files** — no files found.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "**Files** — %d indexed\n\n", len(r.Files))
	b.WriteString("| Path | Language | Nodes | Edges |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, f := range r.Files {
		fmt.Fprintf(&b, "| `%s` | %s | %d | %d |\n", f.Path, f.Language, f.NodeCount, f.EdgeCount)
	}
	return b.String()
}

// renderFileTreeMarkdown ports internal/cli/files.go's printFileTree
// algorithm into internal/query as a string-returning renderer, matching
// this file's Render* convention (strings.Builder, not io.Writer).
// Directory nodes render "name/" then recurse two spaces deeper; leaf
// nodes render "name (language)".
//
// internal/cli/files.go's own printFileTree is left in place and
// unmodified — moving it is out of this plan's scope and would put a CLI
// file in this plan's files_modified, creating a wave conflict with plan
// 02-07. This is a deliberate, bounded package-local duplication, the
// same precedent status.go's shouldSkipStaleDir already set (duplicated
// rather than imported to avoid an unwanted dependency edge —
// internal/query must not import internal/cli).
func renderFileTreeMarkdown(nodes []*FileTreeNode, indent string) string {
	var b strings.Builder
	for _, n := range nodes {
		if n.IsDir {
			fmt.Fprintf(&b, "%s%s/\n", indent, n.Name)
			b.WriteString(renderFileTreeMarkdown(n.Children, indent+"  "))
		} else {
			fmt.Fprintf(&b, "%s%s (%s)\n", indent, n.Name, n.Language)
		}
	}
	return b.String()
}
