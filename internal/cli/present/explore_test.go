package present

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// markdownToPlainContract strips markdown SYNTAX ONLY — every "**", every
// backtick, a leading "> " on any line, and any line whose trimmed
// content is exactly "```go" or "```" — reproducing exactly what hue
// replaces in the styled renderers (D-07). Nothing else changes: wording,
// section order and every other byte survive untouched, which is what
// makes stripANSI(styled) == markdownToPlainContract(plain) a real
// content contract rather than a coincidence.
func markdownToPlainContract(md string) string {
	lines := strings.Split(md, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "```go" || trimmed == "```" {
			continue
		}
		line = strings.TrimPrefix(line, "> ")
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "`", "")
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// TestMarkdownToPlainContract pins the transform itself on a small,
// hand-built 6-line sample covering every one of its four rules.
func TestMarkdownToPlainContract(t *testing.T) {
	md := strings.Join([]string{
		"**Bold Header**",
		"plain text with `code` and more",
		"> a blockquote line",
		"```go",
		"1\tsource line",
		"```",
	}, "\n")
	want := strings.Join([]string{
		"Bold Header",
		"plain text with code and more",
		"a blockquote line",
		"1\tsource line",
	}, "\n")
	if got := markdownToPlainContract(md); got != want {
		t.Errorf("markdownToPlainContract() =\n%q\nwant\n%q", got, want)
	}
}

// explorePlainMarkdown builds the plain expectation for r: the Empty case
// duplicates internal/query/explore.go's unexported exploreZeroResult
// plus render_markdown.go's unexported staleBannerText verbatim (both
// unexported — internal/query stays frozen this phase, D-07); the
// populated case calls the EXPORTED query.RenderExplore directly,
// anchoring the contract to real plain output rather than a hand-typed
// string.
func explorePlainMarkdown(r query.ExploreResult) string {
	if r.Empty {
		banner := ""
		if r.Stale {
			banner = "**⚠ Index may be stale — a sync is pending.**\n\n"
		}
		return fmt.Sprintf("%s**Exploration: %s**\n\nFound 0 symbols across 0 files.\n", banner, r.Query)
	}
	return query.RenderExplore(r.Query, len(r.Groups), r.SymbolCount, r.Groups, r.Blasts, r.Sources, r.Stale, r.SkeletonFiles)
}

// TestRenderExploreStrippedEqualsMarkdownContract pins present.RenderExplore
// to query.RenderExplore's (and exploreZeroResult's) markdown across every
// shape D-07 requires: empty, empty+stale, populated (covering all three
// blast-bullet caller conditions), populated+stale, and populated with a
// skeleton file — plus a positive control that the populated styled output
// actually contains an ANSI escape byte.
func TestRenderExploreStrippedEqualsMarkdownContract(t *testing.T) {
	nodeA := &schema.Node{Id: "a", Name: "Alpha", Kind: "func", FilePath: "pkg/a.go", StartLine: 10, EndLine: 12, Signature: "func Alpha()"}
	nodeB := &schema.Node{Id: "b", Name: "Beta", Kind: "func", FilePath: "pkg/b.go", StartLine: 20, EndLine: 22, Signature: "func Beta()"}
	nodeC := &schema.Node{Id: "c", Name: "Gamma", Kind: "func", FilePath: "pkg/a.go", StartLine: 30, EndLine: 30, Signature: "func Gamma()"}

	groups := []query.ExploreFileGroup{
		{Path: "pkg/a.go", Symbols: []*schema.Node{nodeA, nodeC}},
		{Path: "pkg/b.go", Symbols: []*schema.Node{nodeB}},
	}
	blasts := []query.ExploreBlast{
		{Symbol: nodeA, CallerCount: 2, TestFiles: []string{"pkg/a_test.go"}}, // covering tests
		{Symbol: nodeB, CallerCount: 1},                                       // no covering tests found
		{Symbol: nodeC, CallerCount: 0},                                       // zero callers → no clause
	}
	sources := map[string][]byte{
		"pkg/a.go": []byte("package pkg\n\nfunc Alpha() {}\n\nfunc Gamma() {}\n"),
		"pkg/b.go": []byte("package pkg\n\nfunc Beta() {}\n"),
	}
	populated := query.ExploreResult{Query: "alpha beta", Groups: groups, SymbolCount: 3, Blasts: blasts, Sources: sources}

	nodeD := &schema.Node{Id: "d", Name: "Delta", Kind: "struct", FilePath: "pkg/d.go", StartLine: 5, EndLine: 8, Signature: ""}
	skeleton := query.ExploreResult{
		Query:         "delta",
		Groups:        []query.ExploreFileGroup{{Path: "pkg/d.go", Symbols: []*schema.Node{nodeD}}},
		SymbolCount:   1,
		Blasts:        []query.ExploreBlast{{Symbol: nodeD, CallerCount: 0}},
		Sources:       map[string][]byte{"pkg/d.go": []byte("package pkg\n\ntype Delta struct{}\n")},
		SkeletonFiles: map[string]bool{"pkg/d.go": true},
	}

	populatedStale := populated
	populatedStale.Stale = true

	cases := []struct {
		name string
		r    query.ExploreResult
	}{
		{"empty", query.ExploreResult{Query: "nothing found", Empty: true}},
		{"empty-stale", query.ExploreResult{Query: "nothing found", Empty: true, Stale: true}},
		{"populated", populated},
		{"populated-stale", populatedStale},
		{"populated-skeleton", skeleton},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := RenderExplore(tc.r, NewPalette(true), &buf); err != nil {
				t.Fatalf("RenderExplore: %v", err)
			}
			got := stripANSI(buf.String())
			want := markdownToPlainContract(explorePlainMarkdown(tc.r))
			if got != want {
				t.Errorf("RenderExplore(%s) stripped =\n%q\nwant\n%q", tc.name, got, want)
			}
		})
	}

	t.Run("PositiveControl_StyledOutputContainsANSI", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderExplore(populated, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderExplore: %v", err)
		}
		if !strings.Contains(buf.String(), "\x1b[") {
			t.Error("styled populated explore output has no ANSI escape byte")
		}
	})
}
