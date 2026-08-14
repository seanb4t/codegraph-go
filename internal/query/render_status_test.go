package query

import (
	"regexp"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
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
		"Edges by Kind:",
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
		"**Edges by Kind:**",
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

// --- v0.11.0 Phase 1 (D-01/D-02/D-04): DenseEdgesByKind + edgeCounts + the
// Edges by Kind: / **Edges by Kind:** section ---

// TestDenseEdgesByKindKeySetEqualsRankEdges asserts DenseEdgesByKind's key
// set is EXACTLY RankEdges's key set (checked in both directions) when
// given an empty StatusResult — the two-directional shape TestRankEdges
// (rwr_test.go) already establishes for RankEdges itself, applied here to
// DenseEdgesByKind's output rather than restating the 9 kind strings a
// second time.
func TestDenseEdgesByKindKeySetEqualsRankEdges(t *testing.T) {
	got := DenseEdgesByKind(StatusResult{})

	if len(got) != len(RankEdges) {
		t.Fatalf("DenseEdgesByKind(empty StatusResult) has %d keys, want %d (len(RankEdges)): %v", len(got), len(RankEdges), got)
	}
	for k := range RankEdges {
		if _, ok := got[k]; !ok {
			t.Errorf("DenseEdgesByKind(empty StatusResult) missing RankEdges member %q", k)
		}
	}
	for k := range got {
		if !RankEdges[k] {
			t.Errorf("DenseEdgesByKind(empty StatusResult) has unexpected member %q not in RankEdges", k)
		}
	}
}

// TestDenseEdgesByKindExplicitZero asserts a RankEdges member absent from
// the sparse tally is present in the dense result with an explicit 0 —
// the absent-vs-measured-zero distinction D-04 exists to make
// un-confusable — while a mentioned kind's real count survives unchanged.
func TestDenseEdgesByKindExplicitZero(t *testing.T) {
	r := StatusResult{EdgesByKind: map[string]int64{goextract.RefKindCalls: 5}}
	got := DenseEdgesByKind(r)

	if got[goextract.RefKindCalls] != 5 {
		t.Errorf("DenseEdgesByKind[%q] = %d, want 5 (mentioned kind's count must survive unchanged)", goextract.RefKindCalls, got[goextract.RefKindCalls])
	}
	for k := range RankEdges {
		if k == goextract.RefKindCalls {
			continue
		}
		v, ok := got[k]
		if !ok {
			t.Errorf("DenseEdgesByKind missing unmentioned RankEdges member %q", k)
			continue
		}
		if v != 0 {
			t.Errorf("DenseEdgesByKind[%q] = %d, want explicit 0 (absent from the sparse tally)", k, v)
		}
	}
}

// TestDenseEdgesByKindPreservesUnrankedKind asserts a kind outside
// RankEdges (e.g. "contains", per resolve.go:191) with a non-zero count
// is kept in the dense result — the result is the UNION of RankEdges and
// the sparse tally's keys, so an unranked kind is never silently dropped.
func TestDenseEdgesByKindPreservesUnrankedKind(t *testing.T) {
	r := StatusResult{EdgesByKind: map[string]int64{"contains": 7}}
	got := DenseEdgesByKind(r)

	if got["contains"] != 7 {
		t.Errorf("DenseEdgesByKind[%q] = %d, want 7 (unranked kind must survive, not be dropped)", "contains", got["contains"])
	}
	if len(got) != len(RankEdges)+1 {
		t.Errorf("DenseEdgesByKind has %d keys, want %d (len(RankEdges)+1 for the unranked kind)", len(got), len(RankEdges)+1)
	}
}

// TestDenseEdgesByKindDoesNotMutateInput asserts DenseEdgesByKind returns
// a NEW map and never writes into the caller's r.EdgesByKind.
func TestDenseEdgesByKindDoesNotMutateInput(t *testing.T) {
	in := map[string]int64{goextract.RefKindCalls: 3}
	r := StatusResult{EdgesByKind: in}

	_ = DenseEdgesByKind(r)

	if len(in) != 1 {
		t.Fatalf("DenseEdgesByKind mutated the caller's input map: now has %d keys, want 1: %v", len(in), in)
	}
	if in[goextract.RefKindCalls] != 3 {
		t.Errorf("DenseEdgesByKind mutated the caller's input map value: got %d, want 3", in[goextract.RefKindCalls])
	}
}

// TestRenderStatusTextEdgesByKindSection asserts RenderStatusText emits
// an "Edges by Kind:" section between "Nodes by Kind:" and
// "Files by Language:" — position asserted by index comparison, not mere
// substring presence — and that entries in r.EdgesByKind actually appear.
func TestRenderStatusTextEdgesByKindSection(t *testing.T) {
	r := baseStatusResult()
	r.EdgesByKind = map[string]int64{goextract.RefKindCalls: 5, goextract.RefKindImports: 2}
	out := RenderStatusText(r, "/proj")

	idxNodes := strings.Index(out, "Nodes by Kind:")
	idxEdges := strings.Index(out, "Edges by Kind:")
	idxFiles := strings.Index(out, "Files by Language:")
	if idxNodes < 0 || idxEdges < 0 || idxFiles < 0 {
		t.Fatalf("RenderStatusText output missing one of Nodes by Kind:/Edges by Kind:/Files by Language:\n--- output ---\n%s", out)
	}
	if !(idxNodes < idxEdges && idxEdges < idxFiles) {
		t.Errorf("RenderStatusText section order = Nodes@%d Edges@%d Files@%d, want Nodes < Edges < Files\n--- output ---\n%s", idxNodes, idxEdges, idxFiles, out)
	}
	if !strings.Contains(out, goextract.RefKindCalls) || !strings.Contains(out, goextract.RefKindImports) {
		t.Errorf("RenderStatusText output missing edge-kind rows from r.EdgesByKind\n--- output ---\n%s", out)
	}
}

// TestRenderStatusMarkdownEdgesByKindSection is TestRenderStatusTextEdgesByKindSection's
// MCP-markdown twin: "**Edges by Kind:**" between "**Nodes by Kind:**" and
// "**Languages:**".
func TestRenderStatusMarkdownEdgesByKindSection(t *testing.T) {
	r := baseStatusResult()
	r.EdgesByKind = map[string]int64{goextract.RefKindCalls: 5, goextract.RefKindImports: 2}
	out := RenderStatusMarkdown(r)

	idxNodes := strings.Index(out, "**Nodes by Kind:**")
	idxEdges := strings.Index(out, "**Edges by Kind:**")
	idxLangs := strings.Index(out, "**Languages:**")
	if idxNodes < 0 || idxEdges < 0 || idxLangs < 0 {
		t.Fatalf("RenderStatusMarkdown output missing one of **Nodes by Kind:**/**Edges by Kind:**/**Languages:**\n--- output ---\n%s", out)
	}
	if !(idxNodes < idxEdges && idxEdges < idxLangs) {
		t.Errorf("RenderStatusMarkdown section order = Nodes@%d Edges@%d Languages@%d, want Nodes < Edges < Languages\n--- output ---\n%s", idxNodes, idxEdges, idxLangs, out)
	}
	if !strings.Contains(out, goextract.RefKindCalls) || !strings.Contains(out, goextract.RefKindImports) {
		t.Errorf("RenderStatusMarkdown output missing edge-kind rows from r.EdgesByKind\n--- output ---\n%s", out)
	}
}

// TestRenderStatusEdgeOrderIsDeterministic renders the SAME dense-zero
// StatusResult twice through both renderers and requires byte-identical
// output each time — including a dense map where every RankEdges entry
// ties at 0, which is exactly the case a missing or non-deterministic
// tiebreak would surface as a flake in the byte-frozen wire-oracle
// transcript.
func TestRenderStatusEdgeOrderIsDeterministic(t *testing.T) {
	r := baseStatusResult()
	r.EdgesByKind = DenseEdgesByKind(r) // all RankEdges members present, all tied at 0

	textOut1 := RenderStatusText(r, "/proj")
	textOut2 := RenderStatusText(r, "/proj")
	if textOut1 != textOut2 {
		t.Errorf("RenderStatusText output is not byte-identical across repeated renders of the same dense-zero StatusResult:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", textOut1, textOut2)
	}

	mdOut1 := RenderStatusMarkdown(r)
	mdOut2 := RenderStatusMarkdown(r)
	if mdOut1 != mdOut2 {
		t.Errorf("RenderStatusMarkdown output is not byte-identical across repeated renders of the same dense-zero StatusResult:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", mdOut1, mdOut2)
	}
}
