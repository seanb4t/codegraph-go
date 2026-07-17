package present

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// kindCount pairs a breakdown key (a node kind or a file language) with
// its count — a package-local duplicate of internal/query's unexported
// kindCount (RESEARCH Open Question #1: duplicate rather than import
// internal/query's unexported formatting helpers, matching this
// codebase's existing precedent of package-local duplication across the
// query/cli boundary, e.g. render_results.go's renderFileTreeMarkdown vs
// internal/cli/files.go's printFileTree).
type kindCount struct {
	Key   string
	Count int64
}

// formatNumber is a fixed, en-US-style comma grouper — byte-for-byte
// identical to internal/query's unexported formatNumber (D-02: present
// decorates the plain data, it never re-derives formatting).
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

// formatMB renders bytes as a two-decimal MB value, byte-for-byte
// identical to internal/query's unexported formatMB.
func formatMB(bytes int64) string {
	return fmt.Sprintf("%.2f MB", float64(bytes)/1024/1024)
}

// sortedCounts filters m to count>0 entries and sorts them by count
// DESCENDING, breaking ties on the key ascending — byte-for-byte
// identical to internal/query's unexported sortedCounts.
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

// statLabelWidth/breakdownKeyWidth mirror internal/query's column widths
// (D-09) so the pretty layout keeps the same alignment as the plain form.
const (
	statLabelWidth    = 11
	breakdownKeyWidth = 15
)

// writeStatLine writes one styled "  <label:><value>\n" row of the Index
// Statistics block, label left-justified to statLabelWidth columns and
// rendered via labelStyle (structural chrome only — value is repo-derived
// data passed through unstyled).
func writeStatLine(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "  %s%s\n", labelStyle.Render(fmt.Sprintf("%-*s", statLabelWidth, label+":")), value)
}

// writeBreakdownText writes one styled breakdown section: a sectionStyle
// header line followed by one "  <key padded to 15> <formatNumber(count)>\n"
// row per entry in counts (already filtered/sorted by sortedCounts —
// present never recomputes this).
func writeBreakdownText(b *strings.Builder, header string, counts []kindCount) {
	b.WriteString(sectionStyle.Render(header) + "\n")
	for _, kc := range counts {
		fmt.Fprintf(b, "  %-*s %s\n", breakdownKeyWidth, kc.Key, formatNumber(kc.Count))
	}
}

// writeStatusAdvisories writes the staleness + reindex advisory lines,
// driven by r.Stale and r.Index.ReindexRecommended — the same live
// signals query.RenderStatusText uses (D-02: present decorates, never
// recomputes).
func writeStatusAdvisories(b *strings.Builder, r query.StatusResult, staleLabel, reindexLabel string) {
	if r.Stale {
		fmt.Fprintf(b, "\n%s a sync is recommended — this index may be stale. Run \"codegraph sync\" to update.\n", staleLabel)
	} else {
		b.WriteString("\nIndex is up to date.\n")
	}

	if r.Index.ReindexRecommended {
		fmt.Fprintf(b, "\n%s this index predates the current schema version. Run \"codegraph index --force\" to rebuild.\n", reindexLabel)
	}
}

// RenderStatus writes a lipgloss-styled rendering of r to w, walking the
// SAME section order as query.RenderStatusText (header → Project →
// worktree warning when present → Index Statistics → Nodes by Kind →
// Files by Language → advisories) with headerStyle/labelStyle/sectionStyle
// applied as structural chrome only (D-01/D-02). r is consumed read-only:
// counts, sort order, and wording are never re-derived here — only
// styling is added. Callers gate this behind ChoosePresentation (D-03);
// RenderStatus itself never reads a TTY/env value.
func RenderStatus(r query.StatusResult, projectPath string, w io.Writer) error {
	var b strings.Builder
	b.WriteString(headerStyle.Render("CodeGraph Status") + "\n\n")
	fmt.Fprintf(&b, "%s %s\n", labelStyle.Render("Project:"), projectPath)

	if warning := r.WorktreeMismatch.Warning(); warning != "" {
		b.WriteString(warning + "\n")
	}

	b.WriteString("\n" + sectionStyle.Render("Index Statistics:") + "\n")
	writeStatLine(&b, "Files", formatNumber(r.FileCount))
	writeStatLine(&b, "Nodes", formatNumber(r.NodeCount))
	writeStatLine(&b, "Edges", formatNumber(r.EdgeCount))
	writeStatLine(&b, "DB Size", formatMB(r.DbSizeBytes))
	writeStatLine(&b, "Backend", r.Backend)

	writeBreakdownText(&b, "Nodes by Kind:", sortedCounts(r.NodesByKind))
	writeBreakdownText(&b, "Files by Language:", sortedCounts(r.FilesByLanguage))

	writeStatusAdvisories(&b, r, "Pending Changes:", "Reindex recommended:")

	_, err := io.WriteString(w, b.String())
	return err
}
