package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// sourceDisclaimer is the verbatim-source disclaimer paragraph, copied
// byte-for-byte from testdata/golden/corpus/weft-go/explore.json (D-05a).
// This is an agent-facing contract — reproduce it, do not paraphrase it
// (T-03-06-Drift, RESEARCH Pitfall 3).
const sourceDisclaimer = "The code below is the **verbatim, current on-disk source** of these files — re-read from disk on this call and line-numbered, byte-for-byte identical to what the Read tool returns. It is NOT a summary, outline, or stale cache. Treat each block as a Read you have already performed: do not Read a file shown here."

// staleBannerText is the single bolded line prepended to explore output
// while a sync is pending (D-04a) — exact wording is executor's discretion
// per CONTEXT, kept to one line so it never disturbs the fixed golden
// section order that follows it.
const staleBannerText = "**⚠ Index may be stale — a sync is pending.**\n\n"

// staleBanner returns staleBannerText when stale, or "" when the graph is
// current — the shared prefix both RenderExplore and Explore's zero-match
// early-return literal use, so the banner never has two implementations.
func staleBanner(stale bool) string {
	if !stale {
		return ""
	}
	return staleBannerText
}

// renderNumberedSource renders content as a tab-indented, line-numbered
// fenced "```go" code block (D-05a/D-05b): one "<n>\t<line>" row per source
// line. A trailing newline in content (the common on-disk convention)
// produces no spurious final blank-numbered line — content is split on
// "\n" and a trailing empty final element is dropped before numbering.
func renderNumberedSource(content []byte) string {
	lines := strings.Split(string(content), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	b.WriteString("```go\n")
	for i, line := range lines {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteByte('\t')
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("```\n")
	return b.String()
}

// pluralize returns s unchanged for n==1, else s+"s" — every noun this
// package pluralizes (symbol/file/caller) is regular.
func pluralize(n int, s string) string {
	if n == 1 {
		return s
	}
	return s + "s"
}

// formatNodeRef renders a single "name (path:line)" reference — the
// shared Calls →/Called by ← entry format (D-05b).
func formatNodeRef(n *schema.Node) string {
	return fmt.Sprintf("%s (%s:%d)", n.Name, n.FilePath, n.StartLine)
}

// joinNodeRefs comma-joins formatNodeRef entries, preserving the input
// order (Calls → is forward IterateEdges order; Called by ← is the D-04
// reverse-adjacency map's edge order).
func joinNodeRefs(nodes []*schema.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = formatNodeRef(n)
	}
	return strings.Join(parts, ", ")
}

// RenderNode reproduces the golden node.json markdown shape byte-for-byte
// in its fixed regions (D-05b): "**name** (kind)", a blank line,
// "**Location:**", "**Signature:**", the fixed Trail line, "**Calls →**"
// (forward edges), and "**Called by ←**" (reverse edges) — each entry
// list comma-joined via joinNodeRefs.
func RenderNode(n *schema.Node, calls, calledBy []*schema.Node) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** (%s)\n\n", n.Name, n.Kind)
	fmt.Fprintf(&b, "**Location:** %s:%d\n", n.FilePath, n.StartLine)
	fmt.Fprintf(&b, "**Signature:** `%s`\n", n.Signature)
	b.WriteString("**Trail — codegraph_node any of these to follow it (no Read needed)**\n")
	fmt.Fprintf(&b, "**Calls →** %s\n", joinNodeRefs(calls))
	fmt.Fprintf(&b, "**Called by ←** %s\n", joinNodeRefs(calledBy))
	return b.String()
}

// renderBlastBullet renders one explore.json blast-radius bullet (D-05a):
// "- `name` (path:line) — N caller(s) in `path`" where path is the
// symbol's OWN file (not the callers' files — the golden's "3 callers in
// `internal/cli/finish.go`" counts all 3 callers of mergeStyle, which is
// itself defined in finish.go), plus an optional "; tests: `path`, ..."
// clause listing the distinct files among those callers that are test
// symbols (isTestSymbol, D-07's heuristic).
func renderBlastBullet(bl exploreBlast) string {
	n := bl.Symbol
	s := fmt.Sprintf("- `%s` (%s:%d) — %d %s in `%s`",
		n.Name, n.FilePath, n.StartLine, bl.CallerCount, pluralize(bl.CallerCount, "caller"), n.FilePath)
	if len(bl.TestFiles) > 0 {
		quoted := make([]string, len(bl.TestFiles))
		for i, f := range bl.TestFiles {
			quoted[i] = "`" + f + "`"
		}
		s += "; tests: " + strings.Join(quoted, ", ")
	}
	return s
}

// joinSymbolKindList renders explore's per-file header symbol list:
// "name1(kind1), name2(kind2)" (D-05a's "sym(kind)" per-file header,
// extended to multiple symbols matched in the same file).
func joinSymbolKindList(nodes []*schema.Node) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = fmt.Sprintf("%s(%s)", n.Name, n.Kind)
	}
	return strings.Join(parts, ", ")
}

// RenderExplore reproduces the golden explore.json markdown shape
// byte-for-byte in its fixed regions (D-05a): the exploration header, the
// "Found N symbol(s) across M file(s)." line, the blast-radius bullets,
// the verbatim-source disclaimer, and one "**`path`** — sym(kind), ..."
// header + fenced numbered-source block per matched file, in groups'
// order. When stale is true (D-04a), a single bolded staleness line is
// prepended before the exploration header; a current graph (stale=false)
// prepends nothing, keeping the golden's fixed section order untouched.
func RenderExplore(query string, fileCount, symbolCount int, groups []exploreFileGroup, blasts []exploreBlast, sources map[string][]byte, stale bool) string {
	var b strings.Builder
	b.WriteString(staleBanner(stale))
	fmt.Fprintf(&b, "**Exploration: %s**\n\n", query)
	fmt.Fprintf(&b, "Found %d %s across %d %s.\n\n", symbolCount, pluralize(symbolCount, "symbol"), fileCount, pluralize(fileCount, "file"))
	b.WriteString("**Blast radius — what depends on these (update/verify before editing)**\n\n")
	for _, bl := range blasts {
		b.WriteString(renderBlastBullet(bl))
		b.WriteByte('\n')
	}
	b.WriteString("\n**Source Code**\n\n")
	fmt.Fprintf(&b, "> %s\n\n", sourceDisclaimer)
	for _, g := range groups {
		fmt.Fprintf(&b, "**`%s`** — %s\n\n", g.Path, joinSymbolKindList(g.Symbols))
		b.WriteString(renderNumberedSource(sources[g.Path]))
		b.WriteByte('\n')
	}
	return b.String()
}
