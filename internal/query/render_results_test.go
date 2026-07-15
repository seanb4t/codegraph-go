package query

import (
	"encoding/json"
	"strings"
	"testing"
)

// nonAlphaLocations returns three Location fixtures deliberately NOT in
// alphabetical order by Name ("zeta", "alpha", "mid") — SURF-06's Test 4
// (order preservation) asserts the renderer preserves THIS input order
// exactly, so a stray `sort.Slice`/`sort.Strings` call inside a renderer
// would flip "zeta" after "alpha" and be caught immediately.
func nonAlphaLocations() []Location {
	return []Location{
		{Name: "zeta", Kind: "function", FilePath: "internal/z.go", StartLine: 10},
		{Name: "alpha", Kind: "method", FilePath: "internal/a.go", StartLine: 20},
		{Name: "mid", Kind: "function", FilePath: "internal/m.go", StartLine: 30},
	}
}

// assertNotJSON requires that out does NOT unmarshal as JSON — the
// SURF-06 contract test that makes a silent regression back to
// json.Marshal impossible. This assertion pairs with assertContains
// (a positive markdown-marker check): the negative alone would pass for
// an empty/garbage string, and the positive alone would pass for a JSON
// blob that happens to also contain the marker text — both are required.
func assertNotJSON(t *testing.T, out string) {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(out), &v); err == nil {
		t.Fatalf("expected output to NOT be valid JSON, but it unmarshaled cleanly: %q", out)
	}
}

// assertANSIFree requires out contains no ESC byte (0x1b) — plain-text-only
// constraint; Phase 6 (TUI-02) owns colorization, this phase delivers
// structure and wording only.
func assertANSIFree(t *testing.T, out string) {
	t.Helper()
	if strings.ContainsRune(out, 0x1b) {
		t.Fatalf("expected no ANSI escape byte in output, found one: %q", out)
	}
}

// assertOrder requires first appears before second in out — the order-
// preservation check for the non-alphabetical fixtures.
func assertOrder(t *testing.T, out, first, second string) {
	t.Helper()
	fi := strings.Index(out, first)
	si := strings.Index(out, second)
	if fi == -1 || si == -1 {
		t.Fatalf("expected both %q and %q to appear in output: %q", first, second, out)
	}
	if fi >= si {
		t.Fatalf("expected %q to appear before %q (order preservation), got indices %d >= %d: %q", first, second, fi, si, out)
	}
}

// --- renderLocationTable (unexported, shared by 4 renderers) ---

func TestRenderLocationTable(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		out := renderLocationTable(nil)
		if out != "" {
			t.Fatalf("expected empty string for zero-length slice, got %q", out)
		}
	})

	t.Run("not valid JSON", func(t *testing.T) {
		assertNotJSON(t, renderLocationTable(nonAlphaLocations()))
	})

	t.Run("positive markdown marker", func(t *testing.T) {
		out := renderLocationTable(nonAlphaLocations())
		if !strings.Contains(out, "|") {
			t.Fatalf("expected a markdown table marker (|) in output: %q", out)
		}
	})

	t.Run("order preservation", func(t *testing.T) {
		out := renderLocationTable(nonAlphaLocations())
		assertOrder(t, out, "zeta", "alpha")
		assertOrder(t, out, "alpha", "mid")
	})

	t.Run("single row", func(t *testing.T) {
		out := renderLocationTable([]Location{{Name: "solo", Kind: "function", FilePath: "internal/solo.go", StartLine: 1}})
		if !strings.Contains(out, "solo") {
			t.Fatalf("expected single row to render its Name: %q", out)
		}
	})

	t.Run("ANSI-free", func(t *testing.T) {
		assertANSIFree(t, renderLocationTable(nonAlphaLocations()))
	})
}

// --- RenderCallersMarkdown ---

func TestRenderCallersMarkdown(t *testing.T) {
	t.Run("not valid JSON", func(t *testing.T) {
		out := RenderCallersMarkdown(CallersResult{Symbol: "Alpha", Callers: nonAlphaLocations()})
		assertNotJSON(t, out)
	})

	t.Run("positive markdown marker", func(t *testing.T) {
		out := RenderCallersMarkdown(CallersResult{Symbol: "Alpha", Callers: nonAlphaLocations()})
		if !strings.Contains(out, "Alpha") || !strings.Contains(out, "|") {
			t.Fatalf("expected symbol name and a table marker in output: %q", out)
		}
	})

	t.Run("empty set renders explicit no-results sentence, not a bare table", func(t *testing.T) {
		out := RenderCallersMarkdown(CallersResult{Symbol: "Lonely", Callers: nil})
		if strings.Contains(out, "|") {
			t.Fatalf("expected no table markers in an empty-result render, got: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "no callers") {
			t.Fatalf("expected an explicit no-callers sentence naming the symbol: %q", out)
		}
		if !strings.Contains(out, "Lonely") {
			t.Fatalf("expected the empty-result sentence to name the symbol: %q", out)
		}
	})

	t.Run("order preservation", func(t *testing.T) {
		out := RenderCallersMarkdown(CallersResult{Symbol: "Alpha", Callers: nonAlphaLocations()})
		assertOrder(t, out, "zeta", "alpha")
		assertOrder(t, out, "alpha", "mid")
	})

	t.Run("single and multi row", func(t *testing.T) {
		single := RenderCallersMarkdown(CallersResult{Symbol: "S", Callers: nonAlphaLocations()[:1]})
		if !strings.Contains(single, "zeta") {
			t.Fatalf("expected single-row render to contain its Name: %q", single)
		}
		multi := RenderCallersMarkdown(CallersResult{Symbol: "S", Callers: nonAlphaLocations()})
		for _, name := range []string{"zeta", "alpha", "mid"} {
			if !strings.Contains(multi, name) {
				t.Fatalf("expected multi-row render to contain %q: %q", name, multi)
			}
		}
	})

	t.Run("ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderCallersMarkdown(CallersResult{Symbol: "Alpha", Callers: nonAlphaLocations()}))
	})
}

// --- RenderCalleesMarkdown ---

func TestRenderCalleesMarkdown(t *testing.T) {
	t.Run("not valid JSON", func(t *testing.T) {
		out := RenderCalleesMarkdown(CalleesResult{Symbol: "Alpha", Callees: nonAlphaLocations()})
		assertNotJSON(t, out)
	})

	t.Run("positive markdown marker", func(t *testing.T) {
		out := RenderCalleesMarkdown(CalleesResult{Symbol: "Alpha", Callees: nonAlphaLocations()})
		if !strings.Contains(out, "Alpha") || !strings.Contains(out, "|") {
			t.Fatalf("expected symbol name and a table marker in output: %q", out)
		}
	})

	t.Run("empty set renders explicit no-results sentence, not a bare table", func(t *testing.T) {
		out := RenderCalleesMarkdown(CalleesResult{Symbol: "Lonely", Callees: nil})
		if strings.Contains(out, "|") {
			t.Fatalf("expected no table markers in an empty-result render, got: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "no callees") {
			t.Fatalf("expected an explicit no-callees sentence naming the symbol: %q", out)
		}
		if !strings.Contains(out, "Lonely") {
			t.Fatalf("expected the empty-result sentence to name the symbol: %q", out)
		}
	})

	t.Run("order preservation", func(t *testing.T) {
		out := RenderCalleesMarkdown(CalleesResult{Symbol: "Alpha", Callees: nonAlphaLocations()})
		assertOrder(t, out, "zeta", "alpha")
		assertOrder(t, out, "alpha", "mid")
	})

	t.Run("single and multi row", func(t *testing.T) {
		single := RenderCalleesMarkdown(CalleesResult{Symbol: "S", Callees: nonAlphaLocations()[:1]})
		if !strings.Contains(single, "zeta") {
			t.Fatalf("expected single-row render to contain its Name: %q", single)
		}
		multi := RenderCalleesMarkdown(CalleesResult{Symbol: "S", Callees: nonAlphaLocations()})
		for _, name := range []string{"zeta", "alpha", "mid"} {
			if !strings.Contains(multi, name) {
				t.Fatalf("expected multi-row render to contain %q: %q", name, multi)
			}
		}
	})

	t.Run("ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderCalleesMarkdown(CalleesResult{Symbol: "Alpha", Callees: nonAlphaLocations()}))
	})
}

// --- RenderImpactMarkdown ---

func TestRenderImpactMarkdown(t *testing.T) {
	mkResult := func(affected []Location) ImpactResult {
		return ImpactResult{Symbol: "Alpha", Depth: 2, NodeCount: 5, EdgeCount: 7, Affected: affected}
	}

	t.Run("not valid JSON", func(t *testing.T) {
		assertNotJSON(t, RenderImpactMarkdown(mkResult(nonAlphaLocations())))
	})

	t.Run("positive markdown marker", func(t *testing.T) {
		out := RenderImpactMarkdown(mkResult(nonAlphaLocations()))
		if !strings.Contains(out, "Alpha") || !strings.Contains(out, "|") {
			t.Fatalf("expected symbol name and a table marker in output: %q", out)
		}
	})

	t.Run("header carries depth/nodeCount/edgeCount scalars", func(t *testing.T) {
		out := RenderImpactMarkdown(mkResult(nonAlphaLocations()))
		for _, want := range []string{"2", "5", "7"} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected header to carry scalar %q: %q", want, out)
			}
		}
	})

	t.Run("empty set renders explicit no-results sentence, not a bare table", func(t *testing.T) {
		out := RenderImpactMarkdown(mkResult(nil))
		if strings.Contains(out, "|") {
			t.Fatalf("expected no table markers in an empty-result render, got: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "no ") {
			t.Fatalf("expected an explicit no-results sentence: %q", out)
		}
	})

	t.Run("order preservation", func(t *testing.T) {
		out := RenderImpactMarkdown(mkResult(nonAlphaLocations()))
		assertOrder(t, out, "zeta", "alpha")
		assertOrder(t, out, "alpha", "mid")
	})

	t.Run("single and multi row", func(t *testing.T) {
		single := RenderImpactMarkdown(mkResult(nonAlphaLocations()[:1]))
		if !strings.Contains(single, "zeta") {
			t.Fatalf("expected single-row render to contain its Name: %q", single)
		}
		multi := RenderImpactMarkdown(mkResult(nonAlphaLocations()))
		for _, name := range []string{"zeta", "alpha", "mid"} {
			if !strings.Contains(multi, name) {
				t.Fatalf("expected multi-row render to contain %q: %q", name, multi)
			}
		}
	})

	t.Run("ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderImpactMarkdown(mkResult(nonAlphaLocations())))
	})
}

// --- RenderSearchMarkdown ---

func TestRenderSearchMarkdown(t *testing.T) {
	t.Run("not valid JSON", func(t *testing.T) {
		assertNotJSON(t, RenderSearchMarkdown("needle", nonAlphaLocations()))
	})

	t.Run("positive markdown marker", func(t *testing.T) {
		out := RenderSearchMarkdown("needle", nonAlphaLocations())
		if !strings.Contains(out, "needle") || !strings.Contains(out, "|") {
			t.Fatalf("expected search term and a table marker in output: %q", out)
		}
	})

	t.Run("empty set renders explicit no-results sentence, not a bare table", func(t *testing.T) {
		out := RenderSearchMarkdown("nothingmatches", nil)
		if strings.Contains(out, "|") {
			t.Fatalf("expected no table markers in an empty-result render, got: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "no ") {
			t.Fatalf("expected an explicit no-results sentence: %q", out)
		}
		if !strings.Contains(out, "nothingmatches") {
			t.Fatalf("expected the empty-result sentence to name the search term: %q", out)
		}
	})

	t.Run("order preservation", func(t *testing.T) {
		out := RenderSearchMarkdown("needle", nonAlphaLocations())
		assertOrder(t, out, "zeta", "alpha")
		assertOrder(t, out, "alpha", "mid")
	})

	t.Run("single and multi row", func(t *testing.T) {
		single := RenderSearchMarkdown("needle", nonAlphaLocations()[:1])
		if !strings.Contains(single, "zeta") {
			t.Fatalf("expected single-row render to contain its Name: %q", single)
		}
		multi := RenderSearchMarkdown("needle", nonAlphaLocations())
		for _, name := range []string{"zeta", "alpha", "mid"} {
			if !strings.Contains(multi, name) {
				t.Fatalf("expected multi-row render to contain %q: %q", name, multi)
			}
		}
	})

	t.Run("ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderSearchMarkdown("needle", nonAlphaLocations()))
	})
}

// --- RenderFilesMarkdown ---

func nonAlphaFileEntries() []FileEntry {
	return []FileEntry{
		{Path: "internal/zeta.go", Language: "go", NodeCount: 3, EdgeCount: 5},
		{Path: "internal/alpha.go", Language: "go", NodeCount: 1, EdgeCount: 2},
		{Path: "internal/mid.go", Language: "go", NodeCount: 4, EdgeCount: 6},
	}
}

func TestRenderFilesMarkdown(t *testing.T) {
	t.Run("flat: not valid JSON", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "flat", Files: nonAlphaFileEntries()})
		assertNotJSON(t, out)
	})

	t.Run("flat: positive markdown marker, 4-column table", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "flat", Files: nonAlphaFileEntries()})
		if !strings.Contains(out, "|") {
			t.Fatalf("expected a markdown table marker in output: %q", out)
		}
		for _, want := range []string{"Path", "Language", "Nodes", "Edges"} {
			if !strings.Contains(out, want) {
				t.Fatalf("expected flat table to have a %q column header: %q", want, out)
			}
		}
	})

	t.Run("flat: empty result renders explicit no-files sentence, not a bare table", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "flat", Files: nil})
		if strings.Contains(out, "|") {
			t.Fatalf("expected no table markers in an empty-result render, got: %q", out)
		}
		if !strings.Contains(strings.ToLower(out), "no files") {
			t.Fatalf("expected an explicit no-files sentence: %q", out)
		}
	})

	t.Run("flat: default empty format string is treated as flat", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "", Files: nonAlphaFileEntries()})
		if !strings.Contains(out, "Path") {
			t.Fatalf("expected empty Format to render as the flat table: %q", out)
		}
	})

	t.Run("flat: order preservation", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "flat", Files: nonAlphaFileEntries()})
		assertOrder(t, out, "zeta.go", "alpha.go")
		assertOrder(t, out, "alpha.go", "mid.go")
	})

	t.Run("flat: ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderFilesMarkdown(FilesResult{Format: "flat", Files: nonAlphaFileEntries()}))
	})

	// Tree branch — a UNION with the flat branch; FilesResult.Tree is
	// nested (FileTreeNode), which cannot be represented as a table, per
	// SURF-06's explicit "files --format tree needs a SECOND shape" note.
	tree := []*FileTreeNode{
		{Name: "cmd", IsDir: true, Children: []*FileTreeNode{
			{Name: "main.go", IsDir: false, Path: "cmd/main.go", Language: "go"},
		}},
		{Name: "util.go", IsDir: false, Path: "util.go", Language: "go"},
	}

	t.Run("tree: not valid JSON", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "tree", Tree: tree})
		assertNotJSON(t, out)
	})

	t.Run("tree: renders an indented plain-text list, not a table", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "tree", Tree: tree})
		if strings.Contains(out, "|---") {
			t.Fatalf("expected the tree branch to NOT render a table header separator: %q", out)
		}
		if !strings.Contains(out, "cmd/") {
			t.Fatalf("expected a directory node to render as \"cmd/\": %q", out)
		}
		if !strings.Contains(out, "main.go (go)") {
			t.Fatalf("expected a nested leaf node to render as \"main.go (go)\": %q", out)
		}
		if !strings.Contains(out, "util.go (go)") {
			t.Fatalf("expected a root leaf node to render as \"util.go (go)\": %q", out)
		}
	})

	t.Run("tree: nested leaf is indented deeper than its parent dir", func(t *testing.T) {
		out := RenderFilesMarkdown(FilesResult{Format: "tree", Tree: tree})
		dirIdx := strings.Index(out, "cmd/")
		leafIdx := strings.Index(out, "main.go (go)")
		if dirIdx == -1 || leafIdx == -1 || dirIdx >= leafIdx {
			t.Fatalf("expected \"cmd/\" to appear before its nested \"main.go (go)\": %q", out)
		}
		leafLineStart := strings.LastIndex(out[:leafIdx], "\n") + 1
		dirLineStart := strings.LastIndex(out[:dirIdx], "\n") + 1
		leafIndent := leafIdx - leafLineStart
		dirIndent := dirIdx - dirLineStart
		if leafIndent <= dirIndent {
			t.Fatalf("expected the nested leaf to be indented deeper than its parent dir (leafIndent=%d, dirIndent=%d): %q", leafIndent, dirIndent, out)
		}
	})

	t.Run("tree: ANSI-free", func(t *testing.T) {
		assertANSIFree(t, RenderFilesMarkdown(FilesResult{Format: "tree", Tree: tree}))
	})
}
