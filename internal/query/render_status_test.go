package query

import (
	"regexp"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// baseStatusResult builds a StatusResult fixture in-memory — pure struct
// data, no store/index needed, since RenderStatusText/RenderStatusMarkdown
// are pure functions of StatusResult (plan 02-05 Task 1). The
// FilesByLanguage map deliberately includes a zero-count entry (yaml) and
// non-monotonic counts (go > javascript > python) so a missing count>0
// filter or a wrong sort direction is genuinely caught, not just a
// top-of-list check (a map's Go iteration order is randomized, so a naive
// "top item first" assertion would be flaky rather than reliably wrong —
// this fixture requires asserting the FULL relative order of all three
// surviving keys).
func baseStatusResult() StatusResult {
	return StatusResult{
		Initialized: true,
		FileCount:   100,
		NodeCount:   1234567,
		EdgeCount:   1223,
		DbSizeBytes: 1234567, // ~1.18 MB
		Backend:     "pebble",
		NodesByKind: map[string]int64{
			"function": 10,
			"type":     0,
		},
		FilesByLanguage: map[string]int64{
			"go":         42,
			"python":     7,
			"yaml":       0,
			"javascript": 19,
		},
		Stale: false,
		Index: IndexHealth{
			ReindexRecommended: false,
		},
	}
}

// --- Test 1 (D-10): formatNumber ---

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{1223, "1,223"},
		{1234567, "1,234,567"},
		{-1234, "-1,234"},
	}
	for _, c := range cases {
		if got := formatNumber(c.in); got != c.want {
			t.Errorf("formatNumber(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- Test 2 (D-07): formatMB ---

var mbShapeRE = regexp.MustCompile(`^\d+\.\d{2} MB$`)

func TestFormatMB(t *testing.T) {
	got := formatMB(1234567)
	if !mbShapeRE.MatchString(got) {
		t.Fatalf("formatMB(1234567) = %q, want to match %s", got, mbShapeRE.String())
	}
}

// --- Test 6 (D-09/STAT-02): sortedCounts filter + sort + tiebreak ---

func TestSortedCounts(t *testing.T) {
	m := map[string]int64{
		"go":         42,
		"python":     7,
		"yaml":       0,
		"javascript": 19,
	}
	got := sortedCounts(m)

	wantKeys := []string{"go", "javascript", "python"}
	if len(got) != len(wantKeys) {
		t.Fatalf("sortedCounts(%v) = %v, want %d entries (zero-count key must be dropped)", m, got, len(wantKeys))
	}
	for i, k := range wantKeys {
		if got[i].Key != k {
			t.Errorf("sortedCounts(%v)[%d].Key = %q, want %q (full relative order: %v)", m, i, got[i].Key, k, wantKeys)
		}
	}

	// Tie-break: equal counts break on key ascending (deterministic despite
	// randomized map iteration — TS relies on Object.entries insertion
	// order, which Go cannot reproduce).
	tie := map[string]int64{"zebra": 5, "alpha": 5}
	gotTie := sortedCounts(tie)
	if len(gotTie) != 2 || gotTie[0].Key != "alpha" || gotTie[1].Key != "zebra" {
		t.Errorf("sortedCounts(%v) = %v, want [alpha, zebra] (key-ascending tiebreak)", tie, gotTie)
	}
}

// --- Test 3 (D-09): RenderStatusText section headers + padded labels ---

func TestRenderStatusTextSections(t *testing.T) {
	out := RenderStatusText(baseStatusResult(), "/proj")

	for _, want := range []string{
		"Index Statistics:",
		"Nodes by Kind:",
		"Files by Language:",
		"  Files:     ",
		"  Nodes:     ",
		"  Edges:     ",
		"  DB Size:   ",
		"  Backend:   ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderStatusText output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// --- Test 4 (D-09): Backend renders the Go-truthful value, not TS's string ---

func TestRenderStatusTextBackendFromField(t *testing.T) {
	r := baseStatusResult()
	r.Backend = "pebble"
	out := RenderStatusText(r, "/proj")

	if !strings.Contains(out, "pebble") {
		t.Errorf("RenderStatusText output missing r.Backend value %q\n--- output ---\n%s", r.Backend, out)
	}
	if strings.Contains(out, "node:sqlite") {
		t.Errorf("RenderStatusText output contains TS's hardcoded backend string, want r.Backend only\n--- output ---\n%s", out)
	}
}

// --- Test 5 (D-09): no Journal: line — no Pebble analog ---

func TestRenderStatusTextNoJournalLine(t *testing.T) {
	out := RenderStatusText(baseStatusResult(), "/proj")
	if strings.Contains(out, "Journal:") {
		t.Errorf("RenderStatusText output contains a Journal: line — no Pebble analog exists (D-09)\n--- output ---\n%s", out)
	}
}

// --- Test 6 (D-09/STAT-02): breakdown filter/sort/pad in rendered output ---

func TestRenderStatusTextBreakdownFilterSortPad(t *testing.T) {
	out := RenderStatusText(baseStatusResult(), "/proj")

	if strings.Contains(out, "yaml") {
		t.Errorf("RenderStatusText output contains the zero-count key %q, want it filtered\n--- output ---\n%s", "yaml", out)
	}

	idxGo := strings.Index(out, "go")
	idxJS := strings.Index(out, "javascript")
	idxPy := strings.Index(out, "python")
	if idxGo < 0 || idxJS < 0 || idxPy < 0 {
		t.Fatalf("RenderStatusText output missing one of go/javascript/python\n--- output ---\n%s", out)
	}
	if !(idxGo < idxJS && idxJS < idxPy) {
		t.Errorf("RenderStatusText breakdown order = go@%d javascript@%d python@%d, want ascending (count DESC: go(42) > javascript(19) > python(7))\n--- output ---\n%s", idxGo, idxJS, idxPy, out)
	}

	// padEnd(15) on the key: "go" padded to 15 columns before the count.
	wantGoRow := "  go" + strings.Repeat(" ", 14) + "42"
	if !strings.Contains(out, wantGoRow) {
		t.Errorf("RenderStatusText output missing padded breakdown row %q\n--- output ---\n%s", wantGoRow, out)
	}
}

// --- Test 7 (STAT-03/D-06): staleness advisory driven by Stale, not counts ---

func TestRenderStatusTextStaleAdvisory(t *testing.T) {
	stale := baseStatusResult()
	stale.Stale = true
	outStale := RenderStatusText(stale, "/proj")
	if !strings.Contains(strings.ToLower(outStale), "pending") && !strings.Contains(strings.ToLower(outStale), "stale") {
		t.Errorf("RenderStatusText(Stale=true) output has no pending-sync advisory\n--- output ---\n%s", outStale)
	}

	current := baseStatusResult()
	current.Stale = false
	outCurrent := RenderStatusText(current, "/proj")
	if !strings.Contains(strings.ToLower(outCurrent), "up to date") {
		t.Errorf("RenderStatusText(Stale=false) output has no up-to-date line\n--- output ---\n%s", outCurrent)
	}
	if strings.Contains(strings.ToLower(outCurrent), "pending") {
		t.Errorf("RenderStatusText(Stale=false) output should not carry a pending-sync advisory\n--- output ---\n%s", outCurrent)
	}
}

// --- Test 8 (STAT-03): reindex advisory driven by Index.ReindexRecommended ---

func TestRenderStatusTextReindexAdvisory(t *testing.T) {
	rec := baseStatusResult()
	rec.Index.ReindexRecommended = true
	outRec := RenderStatusText(rec, "/proj")
	if !strings.Contains(strings.ToLower(outRec), "reindex") {
		t.Errorf("RenderStatusText(ReindexRecommended=true) output has no reindex advisory\n--- output ---\n%s", outRec)
	}

	notRec := baseStatusResult()
	notRec.Index.ReindexRecommended = false
	outNotRec := RenderStatusText(notRec, "/proj")
	if strings.Contains(strings.ToLower(outNotRec), "reindex") {
		t.Errorf("RenderStatusText(ReindexRecommended=false) output should not carry a reindex advisory\n--- output ---\n%s", outNotRec)
	}
}

// --- Test 9 (D-11/D-12): verbose worktree warning on RenderStatusText ---

func TestRenderStatusTextWorktreeWarning(t *testing.T) {
	m := &gitmeta.Mismatch{WorktreeRoot: "/w", IndexRoot: "/i"}

	withMismatch := baseStatusResult()
	withMismatch.WorktreeMismatch = m
	out := RenderStatusText(withMismatch, "/proj")
	if !strings.Contains(out, m.Warning()) {
		t.Errorf("RenderStatusText output missing the verbose worktree warning\n--- warning ---\n%s\n--- output ---\n%s", m.Warning(), out)
	}

	noMismatch := baseStatusResult()
	noMismatch.WorktreeMismatch = nil
	outNil := RenderStatusText(noMismatch, "/proj")
	if strings.Contains(outNil, "different git working tree") {
		t.Errorf("RenderStatusText output should not carry a worktree warning when WorktreeMismatch is nil\n--- output ---\n%s", outNil)
	}
}

// --- Test 10 (D-17): RenderStatusMarkdown is structurally different ---

func TestRenderStatusMarkdownShape(t *testing.T) {
	out := RenderStatusMarkdown(baseStatusResult())

	for _, want := range []string{
		"**CodeGraph Status**",
		"**Files indexed:**",
		"**Database size:**",
		"**Nodes by Kind:**",
		"**Languages:**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderStatusMarkdown output missing %q\n--- output ---\n%s", want, out)
		}
	}

	if strings.Contains(out, "Index Statistics:") {
		t.Errorf("RenderStatusMarkdown output contains the CLI's Index Statistics: header — the two renderings must be structurally different (D-17)\n--- output ---\n%s", out)
	}

	// Breakdown bullets use "- key: count" form.
	if !strings.Contains(out, "- go: 42") {
		t.Errorf("RenderStatusMarkdown output missing a %q bullet\n--- output ---\n%s", "- go: 42", out)
	}
}

// --- Test 11 (D-12): MCP form embeds the blockquoted verbose warning ---

func TestRenderStatusMarkdownWorktreeBlockquote(t *testing.T) {
	m := &gitmeta.Mismatch{WorktreeRoot: "/w", IndexRoot: "/i"}
	r := baseStatusResult()
	r.WorktreeMismatch = m
	out := RenderStatusMarkdown(r)

	if !strings.Contains(out, "> ") {
		t.Fatalf("RenderStatusMarkdown output missing a blockquoted line\n--- output ---\n%s", out)
	}

	blockquote := WorktreeWarningBlockquote(m)
	for _, line := range strings.Split(strings.TrimRight(blockquote, "\n"), "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "> ") {
			t.Errorf("blockquote line %q does not start with %q", line, "> ")
		}
	}
	if !strings.Contains(out, blockquote) {
		t.Errorf("RenderStatusMarkdown output does not embed WorktreeWarningBlockquote(m) verbatim\n--- blockquote ---\n%s\n--- output ---\n%s", blockquote, out)
	}
}

// --- Test 12: neither renderer emits an ANSI escape ---

func TestRenderStatusNoANSI(t *testing.T) {
	m := &gitmeta.Mismatch{WorktreeRoot: "/w", IndexRoot: "/i"}
	r := baseStatusResult()
	r.WorktreeMismatch = m
	r.Stale = true
	r.Index.ReindexRecommended = true

	for name, out := range map[string]string{
		"RenderStatusText":     RenderStatusText(r, "/proj"),
		"RenderStatusMarkdown": RenderStatusMarkdown(r),
	} {
		if strings.ContainsRune(out, '\x1b') {
			t.Errorf("%s output contains an ANSI escape byte (0x1b) — Phase 6 owns colorization, not this plan", name)
		}
	}
}
