package query

import (
	"fmt"
	"path"
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

// renderNumberedSourceRange is renderNumberedSource's sibling for
// NODE-02's multi-def body rendering: the documented renderNodeSection shows only
// the definition's own body (its [StartLine,EndLine] span), not its
// enclosing file, and each row keeps its TRUE on-disk line number
// (matching the golden captures, e.g. a definition starting at line 10
// is numbered "10\t...", not renumbered from 1). startLine/endLine are
// 1-indexed and inclusive; a zero or out-of-range endLine clamps to the
// file's last line, and an out-of-range startLine clamps into range, so
// a stale/inconsistent Node record degrades to "as much as fits" rather
// than panicking on a slice out of bounds.
func renderNumberedSourceRange(content []byte, startLine, endLine int32) string {
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

	var b strings.Builder
	b.WriteString("```go\n")
	for i := start; i <= end && i >= 1; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('\t')
		b.WriteString(lines[i-1])
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

// nodeMultiDefHardCap, nodeMultiDefBodyBudget, and nodeMultiDefListCap
// are the documented NODE-02 budget constants (RESEARCH §8, verbatim):
// HARD_CAP bounds how many full bodies are ever rendered, BODY_BUDGET is
// the cumulative char budget those bodies must fit within (the first
// body always renders regardless of budget), and LIST_CAP bounds how
// many overflow definitions are listed inline before an "+K more" tail.
// Together they bound NODE-02's output size (T-01-07, DoS mitigation).
const (
	nodeMultiDefHardCap    = 16
	nodeMultiDefBodyBudget = 12000
	nodeMultiDefListCap    = 20
)

// nodeSectionFetch resolves the full-body render inputs (verbatim
// on-disk source plus the forward/reverse call trail) for one multi-def
// candidate. RenderNodeMultiDef calls this LAZILY, in matches order,
// and stops calling it once HARD_CAP renders have been produced —
// mirroring the documented loop (RESEARCH §8) so a symbol with far more
// definitions than the budget allows never pays the I/O cost of
// resolving bodies past HARD_CAP.
type nodeSectionFetch func(n *schema.Node) (source []byte, calls, calledBy []*schema.Node, err error)

// renderNodeSection renders one multi-def candidate's full detail
// (name/kind, Location, Signature, verbatim source, and — only when
// non-empty — the Trail/Calls/CalledBy lines): the documented
// renderNodeSection(cg, n, true) (RESEARCH §8). source is the
// definition's ENCLOSING FILE's full content — renderNodeSection slices
// it down to n's own [StartLine,EndLine] span (renderNumberedSourceRange),
// matching the golden captures, which show only the definition's body,
// not its whole file. Unlike the single-def RenderNode, the Trail line
// (and its Calls/CalledBy children) is OMITTED ENTIRELY when both calls
// and calledBy are empty — confirmed against the golden captures
// (testdata/golden/corpus/*/node-multi.json; e.g. the behavioral corpus's
// "Validate" case, where both definitions have zero callers and neither
// section renders a "**Called by ←**" line at all, unlike the single-def
// path which always renders it even when empty).
func renderNodeSection(n *schema.Node, source []byte, calls, calledBy []*schema.Node) string {
	parts := []string{
		fmt.Sprintf("**%s** (%s)", n.Name, n.Kind),
		"",
		fmt.Sprintf("**Location:** %s:%d", n.FilePath, n.StartLine),
		fmt.Sprintf("**Signature:** `%s`", n.Signature),
		"",
		strings.TrimSuffix(renderNumberedSourceRange(source, n.StartLine, n.EndLine), "\n"),
	}
	if len(calls) > 0 || len(calledBy) > 0 {
		parts = append(parts, "**Trail — codegraph_node any of these to follow it (no Read needed)**")
		if len(calls) > 0 {
			parts = append(parts, fmt.Sprintf("**Calls →** %s", joinNodeRefs(calls)))
		}
		if len(calledBy) > 0 {
			parts = append(parts, fmt.Sprintf("**Called by ←** %s", joinNodeRefs(calledBy)))
		}
	}
	return strings.Join(parts, "\n")
}

// RenderNodeMultiDef reproduces the documented multi-def node markdown shape
// byte-for-byte (NODE-02, RESEARCH §8, Pitfall 4): the "**N definitions
// named "X"**" header line, immediately followed (single newline, NOT a
// blank line — Pitfall 4) by the "Returning M in full[; K more listed
// below] — pick the one you need (no Read required)." line, a blank
// line, then up to HARD_CAP full bodies (renderNodeSection) joined by
// "\n\n---\n\n" and capped at a BODY_BUDGET char budget (always
// rendering at least the first, regardless of budget). When any
// definitions overflow the cap or budget, a "**Other definitions**"
// list follows — capped at LIST_CAP entries with a trailing "+K more"
// line beyond that — plus a closing "Need one of these in full?" hint.
func RenderNodeMultiDef(symbol string, matches []*schema.Node, fetch nodeSectionFetch) (string, error) {
	var rendered []string
	var listed []*schema.Node
	used := 0
	for _, n := range matches {
		if len(rendered) >= nodeMultiDefHardCap {
			listed = append(listed, n)
			continue
		}
		source, calls, calledBy, err := fetch(n)
		if err != nil {
			return "", err
		}
		section := renderNodeSection(n, source, calls, calledBy)
		if len(rendered) == 0 || used+len(section) <= nodeMultiDefBodyBudget {
			rendered = append(rendered, section)
			used += len(section)
		} else {
			listed = append(listed, n)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "**%d definitions named %q**\n", len(matches), symbol)
	overflowClause := ""
	if len(listed) > 0 {
		overflowClause = fmt.Sprintf("; %d more listed below", len(listed))
	}
	fmt.Fprintf(&b, "Returning %d in full%s — pick the one you need (no Read required).\n\n", len(rendered), overflowClause)
	b.WriteString(strings.Join(rendered, "\n\n---\n\n"))
	b.WriteByte('\n')

	if len(listed) > 0 {
		b.WriteByte('\n')
		b.WriteString("**Other definitions**\n")
		capped := listed
		if len(capped) > nodeMultiDefListCap {
			capped = capped[:nodeMultiDefListCap]
		}
		for _, n := range capped {
			fmt.Fprintf(&b, "- `%s` (%s) — %s:%d\n", n.Name, n.Kind, n.FilePath, n.StartLine)
		}
		if len(listed) > nodeMultiDefListCap {
			fmt.Fprintf(&b, "- … +%d more\n", len(listed)-nodeMultiDefListCap)
		}
		b.WriteByte('\n')
		fmt.Fprintf(&b, "> Need one of these in full? Call codegraph_node again with `file` (e.g. `%q`) or `line` — do NOT Read it.\n", path.Base(listed[0].FilePath))
	}

	return b.String(), nil
}

// renderBlastBullet renders one explore.json blast-radius bullet (D-05a):
// "- `name` (path:line) — N caller(s) in `path`" where path is the
// symbol's OWN file (not the callers' files — the golden's "3 callers in
// `internal/cli/finish.go`" counts all 3 callers of mergeStyle, which is
// itself defined in finish.go), plus one of two mutually-exclusive
// trailing clauses (EXPL-04, RESEARCH §5): "; tests: `path`, ..." listing
// the distinct files among those callers that are test symbols
// (isTestSymbol, D-07's heuristic) when at least one exists, or — when the
// root has direct callers but NONE of them are covered by a test — the
// EXACT string "; ⚠️ no covering tests found" (verbatim, note "found", the
// emoji, and that it is appended, not a standalone line). A root with
// ZERO callers gets NEITHER clause, mirroring the documented early-continue before
// this block is ever reached for a caller-less root.
func renderBlastBullet(bl exploreBlast) string {
	n := bl.Symbol
	s := fmt.Sprintf("- `%s` (%s:%d) — %d %s in `%s`",
		n.Name, n.FilePath, n.StartLine, bl.CallerCount, pluralize(bl.CallerCount, "caller"), n.FilePath)
	switch {
	case len(bl.TestFiles) > 0:
		quoted := make([]string, len(bl.TestFiles))
		for i, f := range bl.TestFiles {
			quoted[i] = "`" + f + "`"
		}
		s += "; tests: " + strings.Join(quoted, ", ")
	case bl.CallerCount > 0:
		s += "; ⚠️ no covering tests found"
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

// minPolymorphicSiblings is H20's exact, cited threshold (RESEARCH
// §C.2/H20, mcp/tools.js:2927-2939): an off-spine file whose classes
// share a supertype with >= 3 total implementers renders as a
// signature-only skeleton instead of full source.
const minPolymorphicSiblings = 3

// computeSkeletonFiles is H20 (RESEARCH §C.2, mcp/tools.js:2927-2939):
// determines which selected files should render as signature-only
// skeletons instead of full source. An "on-spine" file (centralFiles,
// H19's central-file selection) NEVER skeletonizes — H20 only applies to
// off-spine files. An off-spine file skeletonizes when at least one of
// its shown symbols implements an interface that has
// >= minPolymorphicSiblings (3) TOTAL implementers (a "polymorphic
// sibling" cluster too large to usefully show every member's full body).
// implementsIdx is BuildImplementsIndex's output (traverse.go, interface
// id -> its implementer edges); implementedInterfaces (traverse.go)
// inverts it to the struct-id -> []interfaceID view this function walks.
func computeSkeletonFiles(groups []exploreFileGroup, centralFiles map[string]bool, implementsIdx map[string][]*schema.Edge) map[string]bool {
	skeleton := make(map[string]bool)
	typeIfaces := implementedInterfaces(implementsIdx)
	for _, g := range groups {
		if centralFiles[g.Path] {
			continue // on-spine files never skeletonize
		}
		for _, n := range g.Symbols {
			for _, ifaceID := range typeIfaces[n.Id] {
				if len(implementsIdx[ifaceID]) >= minPolymorphicSiblings {
					skeleton[g.Path] = true
				}
			}
		}
	}
	return skeleton
}

// renderSkeleton renders H20's signature-only view of symbols: one
// "<kind> <name><signature>" line per symbol inside a fenced code block —
// plain text, no ANSI, matching render_markdown.go's package-wide
// constraint (Phase 6 archtest will enforce it later).
func renderSkeleton(symbols []*schema.Node) string {
	var b strings.Builder
	b.WriteString("```go\n")
	for _, n := range symbols {
		fmt.Fprintf(&b, "%s %s%s\n", n.Kind, n.Name, n.Signature)
	}
	b.WriteString("```\n")
	return b.String()
}

// RenderExplore reproduces the golden explore.json markdown shape
// byte-for-byte in its fixed regions (D-05a): the exploration header, the
// "Found N symbol(s) across M file(s)." line, the blast-radius bullets,
// the verbatim-source disclaimer, and one "**`path`** — sym(kind), ..."
// header + fenced source block per matched file, in groups' order. When
// stale is true (D-04a), a single bolded staleness line is prepended
// before the exploration header; a current graph (stale=false) prepends
// nothing, keeping the golden's fixed section order untouched.
// skeletonFiles (H20, may be nil) renders a file's SIGNATURES ONLY
// (renderSkeleton) instead of its full verbatim source
// (renderNumberedSource) — an off-spine file whose classes share a
// >=3-implementer supertype.
func RenderExplore(query string, fileCount, symbolCount int, groups []exploreFileGroup, blasts []exploreBlast, sources map[string][]byte, stale bool, skeletonFiles map[string]bool) string {
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
		if skeletonFiles[g.Path] {
			b.WriteString(renderSkeleton(g.Symbols))
		} else {
			b.WriteString(renderNumberedSource(sources[g.Path]))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
