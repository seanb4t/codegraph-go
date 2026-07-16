package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TS ships TWO structurally different status renderings — this file
// implements both (D-17):
//
//   - RenderStatusText — the CLI's padded-column layout ("Index Statistics:"
//     / "  Files:     1,234" / padEnd(15) breakdowns), ported from
//     bin/codegraph.js ~900-985 (D-09). Called ONLY by internal/cli.
//   - RenderStatusMarkdown — the MCP's bolded-key bullet layout
//     ("**CodeGraph Status**" / "**Files indexed:** N" / "- kind: count"
//     bullets), ported from mcp/tools.js ~3890-3945. Called ONLY by
//     internal/mcp.
//
// They share the same StatusResult data (STAT-01/02/03) but NOT a
// renderer — TS deliberately ships two shapes for the same data, and
// D-12's blockquote warning only makes sense against the markdown one. A
// future reader collapsing these into a single renderer will break
// TestRenderStatusMarkdownShape's "does not contain Index Statistics:"
// assertion immediately (T-02-22).
//
// Neither renderer may acquire ANSI/color: Phase 6 (TUI-02) owns
// colorization and will add its own unsuffixed, lipgloss-styled
// renderers — these stay plain, stable, agent-parseable text, mirroring
// RenderExplore/RenderNode's convention (render_markdown.go).

// kindCount pairs a breakdown key (a node kind or a file language) with
// its count. sortedCounts produces a slice of these for both
// StatusResult.NodesByKind and StatusResult.FilesByLanguage — the two
// breakdowns share one filter/sort implementation (D-09/D-17), never
// duplicated per-renderer.
type kindCount struct {
	Key   string
	Count int64
}

// formatNumber is a fixed, en-US-style comma grouper (D-10). TS uses
// n.toLocaleString(), which is LOCALE-DEPENDENT: "1,223" under an en-US
// locale, "1.223" or "1 223" under others — and Go has no stdlib
// equivalent. golang.org/x/text/message is explicitly REJECTED: it would
// be a new dependency AND it re-introduces the exact locale variance this
// helper exists to eliminate (Go's message.Printer is locale-neutral by
// default and would still need explicit language.AmericanEnglish
// configuration to match TS's CI behavior). This is a documented,
// intentional Phase-1 D-02 allowed divergence: codegraph-go pins en-US
// grouping everywhere TS follows the host locale.
func formatNumber(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	out := strings.Join(parts, ",")
	if neg {
		out = "-" + out
	}
	return out
}

// formatMB renders bytes as a two-decimal MB value (D-07), matching TS's
// (stats.dbSizeBytes / 1024 / 1024).toFixed(2).
func formatMB(bytes int64) string {
	return fmt.Sprintf("%.2f MB", float64(bytes)/1024/1024)
}

// sortedCounts filters m to count>0 entries and sorts them by count
// DESCENDING, breaking ties on the key ascending. TS relies on
// Object.entries insertion order for ties, which Go cannot reproduce
// (map iteration is deliberately randomized) — a key tiebreak is the
// deterministic substitute, a documented minor divergence. Shared by both
// renderers' Nodes-by-Kind and Files-by-Language/Languages breakdowns.
func sortedCounts(m map[string]int64) []kindCount {
	out := make([]kindCount, 0, len(m))
	for k, v := range m {
		if v > 0 {
			out = append(out, kindCount{Key: k, Count: v})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// statLabelWidth/breakdownKeyWidth are the CLI form's column widths
// (D-09): a stat label ("Files:", "DB Size:") is left-padded to 11
// columns before its value; a breakdown key is left-padded to 15 columns
// before its count — TS's padEnd(15).
const (
	statLabelWidth    = 11
	breakdownKeyWidth = 15
)

// writeStatLine writes one "  <label:><value>\n" row of the CLI's Index
// Statistics: block, label left-justified to statLabelWidth columns.
func writeStatLine(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "  %-*s%s\n", statLabelWidth, label+":", value)
}

// writeBreakdownText writes one CLI-form breakdown section: a header line
// followed by one "  <key padded to 15> <formatNumber(count)>\n" row per
// entry in counts (already filtered/sorted by sortedCounts).
func writeBreakdownText(b *strings.Builder, header string, counts []kindCount) {
	b.WriteString(header + "\n")
	for _, kc := range counts {
		fmt.Fprintf(b, "  %-*s %s\n", breakdownKeyWidth, kc.Key, formatNumber(kc.Count))
	}
}

// writeStatusAdvisories writes the shared staleness + reindex advisory
// lines both renderers append after their breakdowns (STAT-03/D-06).
// ★ Driven by r.Stale and r.Index.ReindexRecommended — the LIVE signals —
// NEVER by the added/modified/removed count breakdown, which stays an
// inert all-zero placeholder per the REQUIREMENTS.md Out-of-Scope table:
// computing exact counts would require re-running Sync's diff on every
// status call. This substitution (TS branches on a real change-list
// length; codegraph-go branches on the live stale bool) is the single
// most likely thing for a future reader to "fix" wrongly — it is not a
// TODO, it is v1.0's deliberate bar.
//
// staleLabel selects between the plain-text CLI wording and the
// bolded-key markdown wording so both renderers keep their own voice
// (D-09 vs D-17) while sharing this one advisory implementation.
func writeStatusAdvisories(b *strings.Builder, r StatusResult, staleLabel, reindexLabel string) {
	if r.Stale {
		fmt.Fprintf(b, "\n%s a sync is recommended — this index may be stale. Run \"codegraph sync\" to update.\n", staleLabel)
	} else {
		b.WriteString("\nIndex is up to date.\n")
	}

	if r.Index.ReindexRecommended {
		fmt.Fprintf(b, "\n%s this index predates the current schema version. Run \"codegraph index --force\" to rebuild.\n", reindexLabel)
	}
}

// RenderStatusText renders StatusResult in TS's CLI padded-column shape
// (D-09), ported from bin/codegraph.js ~900-985. Called ONLY by
// internal/cli (plan 02-07) — see this file's header comment for why a
// second, structurally different renderer exists for MCP.
//
// projectPath is caller-supplied rather than read from r: StatusResult's
// own ProjectPath field is deliberately blanked per its decision table's
// host-path privacy stance (status.go), but the CLI knows its own start
// path and passes it through here for the "Project: <path>" line — the
// same accepted host-path exception WorktreeMismatch already documents.
//
// Journal: is deliberately DROPPED — no Pebble analog exists, consistent
// with the existing journalMode drop in StatusResult's decision table.
// Backend: renders r.Backend (the Go-truthful "pebble"), never a
// hardcoded TS string. TS's indexState (indexing/partial/failed) and
// pendingRefs>0 warning branches are deliberately NOT ported: Go's
// IndexHealth.State is only ever "complete"/"not_indexed" and PendingRefs
// is hard-pinned to 0, so those branches could never fire — porting them
// would be dead code (RESEARCH Pitfall 4).
func RenderStatusText(r StatusResult, projectPath string) string {
	var b strings.Builder
	b.WriteString("CodeGraph Status\n\n")
	fmt.Fprintf(&b, "Project: %s\n", projectPath)

	if warning := r.WorktreeMismatch.Warning(); warning != "" {
		b.WriteString(warning + "\n")
	}

	b.WriteString("\nIndex Statistics:\n")
	writeStatLine(&b, "Files", formatNumber(r.FileCount))
	writeStatLine(&b, "Nodes", formatNumber(r.NodeCount))
	writeStatLine(&b, "Edges", formatNumber(r.EdgeCount))
	writeStatLine(&b, "DB Size", formatMB(r.DbSizeBytes))
	writeStatLine(&b, "Backend", r.Backend)

	writeBreakdownText(&b, "Nodes by Kind:", sortedCounts(r.NodesByKind))
	writeBreakdownText(&b, "Files by Language:", sortedCounts(r.FilesByLanguage))

	writeStatusAdvisories(&b, r, "Pending Changes:", "Reindex recommended:")

	return b.String()
}

// writeBreakdownMarkdown writes one MCP-form breakdown section: a bolded
// header line followed by one "- <key>: <formatNumber(count)>\n" bullet
// per entry in counts (already filtered/sorted by sortedCounts).
func writeBreakdownMarkdown(b *strings.Builder, header string, counts []kindCount) {
	fmt.Fprintf(b, "\n%s\n", header)
	for _, kc := range counts {
		fmt.Fprintf(b, "- %s: %s\n", kc.Key, formatNumber(kc.Count))
	}
}

// RenderStatusMarkdown renders StatusResult in TS's MCP bolded-key
// bullet-list shape (D-17), ported from mcp/tools.js ~3890-3945. Called
// ONLY by internal/mcp (plan 02-06) — see this file's header comment for
// why a second, structurally different renderer exists for the CLI
// (RenderStatusText).
//
// The verbose worktree warning is embedded via WorktreeWarningBlockquote
// (worktree_notice.go) rather than a second inline "\n" -> "\n> "
// transform — one implementation, one place to be wrong (D-12).
//
// DROPPED, and why: TS's "**Journal mode:**" line has no Pebble analog
// (same rationale as RenderStatusText); TS's "**Pending resolution:**" /
// pendingRefs>0 branch is dead code here since PendingRefs is hard-pinned
// to 0 (RESEARCH Pitfall 4); TS's "**Auto-sync disabled:**"
// (isWatcherDegraded) and per-file freshness (getPendingFiles) sections
// both depend on a live watcher, which is Phase 3 (WATCH-01) — not Phase
// 2 content, and are NOT stubbed here.
//
// Note TS labels the FilesByLanguage breakdown "**Languages:**" in this
// form while its CLI labels the identical data "Files by Language:" —
// each surface keeps its own TS-verbatim label.
func RenderStatusMarkdown(r StatusResult) string {
	var b strings.Builder
	b.WriteString("**CodeGraph Status**\n\n")

	if bq := WorktreeWarningBlockquote(r.WorktreeMismatch); bq != "" {
		b.WriteString(bq + "\n\n")
	}

	fmt.Fprintf(&b, "**Files indexed:** %s\n", formatNumber(r.FileCount))
	fmt.Fprintf(&b, "**Total nodes:** %s\n", formatNumber(r.NodeCount))
	fmt.Fprintf(&b, "**Total edges:** %s\n", formatNumber(r.EdgeCount))
	fmt.Fprintf(&b, "**Database size:** %s\n", formatMB(r.DbSizeBytes))
	fmt.Fprintf(&b, "**Backend:** %s\n", r.Backend)

	writeBreakdownMarkdown(&b, "**Nodes by Kind:**", sortedCounts(r.NodesByKind))
	writeBreakdownMarkdown(&b, "**Languages:**", sortedCounts(r.FilesByLanguage))

	writeStatusAdvisories(&b, r, "**Pending Changes:**", "**Reindex recommended:**")

	return b.String()
}
