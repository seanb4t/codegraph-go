package present

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// plainNumbered mirrors internal/query/render_markdown.go's unexported
// renderNumberedSource verbatim — the plain side of the File-mode
// contract (that function is not exported; internal/query stays frozen,
// D-07).
func plainNumbered(content []byte) string {
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

// synthNode builds a synthetic *schema.Node plus a source file body of
// exactly bodyLines filler lines — used to control a multi-def section's
// rendered length precisely enough to cross the BODY_BUDGET/LIST_CAP
// boundaries deliberately.
func synthNode(i, bodyLines int) (*schema.Node, []byte) {
	name := fmt.Sprintf("Sym%d", i)
	filePath := fmt.Sprintf("pkg/file%d.go", i)
	var b strings.Builder
	fmt.Fprintf(&b, "func %s() {\n", name)
	for j := 0; j < bodyLines; j++ {
		fmt.Fprintf(&b, "\tline%04d\n", j)
	}
	b.WriteString("}\n")
	source := []byte(b.String())
	totalLines := bodyLines + 2 // "func ... {" line + body + "}"
	n := &schema.Node{
		Id:        fmt.Sprintf("id%d", i),
		Name:      name,
		Kind:      "func",
		FilePath:  filePath,
		StartLine: 1,
		EndLine:   int32(totalLines),
		Signature: "func()",
	}
	return n, source
}

// multiDefFixture is a shared data source for both the plain
// (query.RenderNodeMultiDef's nodeSectionFetch shape) and styled
// (query.NewMultiDefDetail's definition-func shape) multi-def render
// paths, so both sides render from IDENTICAL underlying data.
type multiDefFixture struct {
	nodes   []*schema.Node
	sources map[string][]byte
}

func newMultiDefFixture(count, bodyLinesForFirst int, hugeFirst bool) *multiDefFixture {
	f := &multiDefFixture{sources: map[string][]byte{}}
	for i := 0; i < count; i++ {
		lines := 2
		if i == 0 && hugeFirst {
			lines = bodyLinesForFirst
		}
		n, src := synthNode(i, lines)
		f.nodes = append(f.nodes, n)
		f.sources[n.Id] = src
	}
	return f
}

// fetchForQuery adapts the fixture to query.RenderNodeMultiDef's
// nodeSectionFetch shape (a func literal is assignable to that unexported
// type, per its own doc comment), counting invocations via counter so the
// test can assert the loop's lazy, in-match-order fetch behaviour.
func (f *multiDefFixture) fetchForQuery(counter *int) func(*schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
	return func(n *schema.Node) ([]byte, []*schema.Node, []*schema.Node, error) {
		*counter++
		return f.sources[n.Id], nil, nil, nil
	}
}

// fetchForDetail adapts the SAME fixture data to
// query.NewMultiDefDetail's definition-func shape, so the styled render
// path fetches from identical data via the identical lazy protocol.
func (f *multiDefFixture) fetchForDetail(counter *int) func(*schema.Node) (*query.DefinitionDetail, error) {
	return func(n *schema.Node) (*query.DefinitionDetail, error) {
		*counter++
		return &query.DefinitionDetail{Node: n, Source: f.sources[n.Id]}, nil
	}
}

// TestRenderNodeStrippedEqualsMarkdownContract pins present.RenderNode to
// query.RenderNode/RenderNodeMultiDef's markdown across all three
// NodeDetailModes and every NODE-02 multi-def budget boundary (HARD_CAP,
// BODY_BUDGET, LIST_CAP), plus a targeted control-byte-drop check.
func TestRenderNodeStrippedEqualsMarkdownContract(t *testing.T) {
	t.Run("File", func(t *testing.T) {
		content := []byte("func\tFaçade() {}\n// comment\n")
		d := query.NodeDetail{Mode: query.NodeDetailModeFile, File: &query.FileDetail{Path: "pkg/x.go", Source: content}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(plainNumbered(content))
		if got != want {
			t.Errorf("File mode stripped =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("SingleDef", func(t *testing.T) {
		n := &schema.Node{Id: "n1", Name: "Handle", Kind: "func", FilePath: "pkg/h.go", StartLine: 7, Signature: "func Handle(w http.ResponseWriter)"}
		caller := &schema.Node{Id: "c1", Name: "Caller", Kind: "func", FilePath: "pkg/c.go", StartLine: 3}
		callee := &schema.Node{Id: "e1", Name: "Callee", Kind: "func", FilePath: "pkg/e.go", StartLine: 9}
		d := query.NodeDetail{Mode: query.NodeDetailModeSingleDef, Definition: &query.DefinitionDetail{Node: n, Calls: []*schema.Node{callee}, CalledBy: []*schema.Node{caller}}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(query.RenderNode(n, []*schema.Node{callee}, []*schema.Node{caller}))
		if got != want {
			t.Errorf("SingleDef stripped =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("SingleDef-nil-refs", func(t *testing.T) {
		n := &schema.Node{Id: "n2", Name: "Lonely", Kind: "func", FilePath: "pkg/l.go", StartLine: 1, Signature: "func Lonely()"}
		d := query.NodeDetail{Mode: query.NodeDetailModeSingleDef, Definition: &query.DefinitionDetail{Node: n}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(query.RenderNode(n, nil, nil))
		if got != want {
			t.Errorf("SingleDef-nil-refs stripped =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("MultiDef-2", func(t *testing.T) {
		f := newMultiDefFixture(2, 2, false)
		var plainCount, styledCount int
		plainOut, err := query.RenderNodeMultiDef("Sym", f.nodes, f.fetchForQuery(&plainCount))
		if err != nil {
			t.Fatalf("query.RenderNodeMultiDef: %v", err)
		}
		multi := query.NewMultiDefDetail("Sym", f.nodes, f.fetchForDetail(&styledCount))
		d := query.NodeDetail{Mode: query.NodeDetailModeMultiDef, Multi: multi}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(plainOut)
		if got != want {
			t.Errorf("MultiDef-2 stripped =\n%q\nwant\n%q", got, want)
		}
		if styledCount != plainCount {
			t.Errorf("MultiDef-2 fetch count: styled=%d plain=%d", styledCount, plainCount)
		}
	})

	t.Run("MultiDef-HardCap-17", func(t *testing.T) {
		f := newMultiDefFixture(17, 2, false)
		var plainCount, styledCount int
		plainOut, err := query.RenderNodeMultiDef("Sym", f.nodes, f.fetchForQuery(&plainCount))
		if err != nil {
			t.Fatalf("query.RenderNodeMultiDef: %v", err)
		}
		multi := query.NewMultiDefDetail("Sym", f.nodes, f.fetchForDetail(&styledCount))
		d := query.NodeDetail{Mode: query.NodeDetailModeMultiDef, Multi: multi}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(plainOut)
		if got != want {
			t.Errorf("MultiDef-HardCap-17 stripped =\n%q\nwant\n%q", got, want)
		}
		if plainCount != nodeMultiDefHardCap {
			t.Errorf("plain fetch count = %d, want %d (hard cap)", plainCount, nodeMultiDefHardCap)
		}
		if styledCount > nodeMultiDefHardCap {
			t.Errorf("styled fetch count = %d, want <= %d (hard cap, lazy, in match order)", styledCount, nodeMultiDefHardCap)
		}
	})

	t.Run("MultiDef-BodyBudget", func(t *testing.T) {
		f := newMultiDefFixture(2, 900, true) // node 0's body alone exceeds nodeMultiDefBodyBudget
		var plainCount, styledCount int
		plainOut, err := query.RenderNodeMultiDef("Sym", f.nodes, f.fetchForQuery(&plainCount))
		if err != nil {
			t.Fatalf("query.RenderNodeMultiDef: %v", err)
		}
		if !strings.Contains(plainOut, "1 more listed below") {
			t.Fatalf("fixture did not exercise BODY_BUDGET overflow: %q", plainOut)
		}
		multi := query.NewMultiDefDetail("Sym", f.nodes, f.fetchForDetail(&styledCount))
		d := query.NodeDetail{Mode: query.NodeDetailModeMultiDef, Multi: multi}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(plainOut)
		if got != want {
			t.Errorf("MultiDef-BodyBudget stripped =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("MultiDef-ListCap-40", func(t *testing.T) {
		f := newMultiDefFixture(40, 900, true) // node 0 huge, so all 39 others overflow BODY_BUDGET
		var plainCount, styledCount int
		plainOut, err := query.RenderNodeMultiDef("Sym", f.nodes, f.fetchForQuery(&plainCount))
		if err != nil {
			t.Fatalf("query.RenderNodeMultiDef: %v", err)
		}
		if !strings.Contains(plainOut, "+19 more") {
			t.Fatalf("fixture did not exercise LIST_CAP overflow to +19 more: %q", plainOut)
		}
		multi := query.NewMultiDefDetail("Sym", f.nodes, f.fetchForDetail(&styledCount))
		d := query.NodeDetail{Mode: query.NodeDetailModeMultiDef, Multi: multi}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		want := markdownToPlainContract(plainOut)
		if got != want {
			t.Errorf("MultiDef-ListCap-40 stripped =\n%q\nwant\n%q", got, want)
		}
		if plainCount != styledCount {
			t.Errorf("fetch count mismatch: plain=%d styled=%d", plainCount, styledCount)
		}
	})

	t.Run("ControlBytesStrippedFromStyled", func(t *testing.T) {
		n := &schema.Node{Id: "cb", Name: "Bad\x1b[31mName", Kind: "func", FilePath: "pkg/cb.go", StartLine: 1, Signature: "func()"}
		d := query.NodeDetail{Mode: query.NodeDetailModeSingleDef, Definition: &query.DefinitionDetail{Node: n}}
		var buf bytes.Buffer
		if err := RenderNode(d, NewPalette(true), &buf); err != nil {
			t.Fatalf("RenderNode: %v", err)
		}
		got := stripANSI(buf.String())
		if strings.ContainsRune(got, 0x1b) {
			t.Errorf("stripped output still contains a raw control byte: %q", got)
		}
	})
}
