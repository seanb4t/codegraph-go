package present

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// stripANSI is defined in ansistrip_test.go (plan 03) — the one shared
// ANSI stripper every renderer contract test in this package reuses.
// This file MUST NOT redeclare it.

// ---------------------------------------------------------------------
// Location fixtures (search/callers/callees/impact/affected all consume
// []query.Location; each renderer's own fixture set below composes these).
// ---------------------------------------------------------------------

func locsEmpty() []query.Location { return nil }

func locsSingle() []query.Location {
	return []query.Location{
		{Name: "Alpha", Kind: "function", FilePath: "pkga/pkga.go", StartLine: 10},
	}
}

// locsThreeUnordered is deliberately NOT alphabetical (Zeta, Alpha, Mu) —
// pins that the styled path preserves engine order (CLI-01 probe ordering).
func locsThreeUnordered() []query.Location {
	return []query.Location{
		{Name: "Zeta", Kind: "function", FilePath: "pkgz/z.go", StartLine: 30},
		{Name: "Alpha", Kind: "method", FilePath: "pkga/a.go", StartLine: 10},
		{Name: "Mu", Kind: "struct", FilePath: "pkgm/m.go", StartLine: 20},
	}
}

// locsDuplicate has two byte-identical entries — pins that the styled
// path never merges/dedupes adjacent identical rows (CLI-01 probe
// adjacency).
func locsDuplicate() []query.Location {
	l := query.Location{Name: "Dup", Kind: "function", FilePath: "pkgd/d.go", StartLine: 5}
	return []query.Location{l, l}
}

// locsUnicode carries non-ASCII bytes in Name/FilePath and, separately, an
// embedded control byte (ESC + newline) in Name — pins CLI-01 probe
// encoding (byte-for-byte passthrough except sanitizeControl) and the
// control-stripping contract in one fixture per requirement.
func locsUnicode() []query.Location {
	return []query.Location{
		{Name: "ünïcode∑", Kind: "function", FilePath: "pkgü/ünïcode.go", StartLine: 7},
	}
}

// locsControl carries a Name with an embedded ESC sequence and a newline —
// the sanitizeControl fixture. The plain expectation is built from the
// SANITIZED name, matching what a real renderer must produce.
func locsControl() []query.Location {
	return []query.Location{
		{Name: "Evil\x1b[31mRed\nName", Kind: "function", FilePath: "pkge/e.go", StartLine: 1},
	}
}

// ---------------------------------------------------------------------
// schema.Node fixtures (RenderSearchFull only)
// ---------------------------------------------------------------------

func nodesEmpty() []*schema.Node { return nil }

func nodesSingle() []*schema.Node {
	return []*schema.Node{
		{Name: "Alpha", Kind: "function", FilePath: "pkga/pkga.go", StartLine: 10, QualifiedName: "pkga.Alpha", Signature: "func()"},
	}
}

func nodesThreeUnordered() []*schema.Node {
	return []*schema.Node{
		{Name: "Zeta", Kind: "function", FilePath: "pkgz/z.go", StartLine: 30, QualifiedName: "pkgz.Zeta", Signature: "func() error"},
		{Name: "Alpha", Kind: "method", FilePath: "pkga/a.go", StartLine: 10, QualifiedName: "pkga.Alpha", Signature: ""},
		{Name: "Mu", Kind: "struct", FilePath: "pkgm/m.go", StartLine: 20, QualifiedName: "pkgm.Mu", Signature: "struct{}"},
	}
}

func nodesDuplicate() []*schema.Node {
	n := &schema.Node{Name: "Dup", Kind: "function", FilePath: "pkgd/d.go", StartLine: 5, QualifiedName: "pkgd.Dup", Signature: "func()"}
	return []*schema.Node{n, n}
}

// nodesUnicode carries the exact non-ASCII signature named in the plan's
// behavior spec.
func nodesUnicode() []*schema.Node {
	return []*schema.Node{
		{Name: "ünïcode∑", Kind: "function", FilePath: "pkgü/ünïcode.go", StartLine: 7, QualifiedName: "pkgü.ünïcode∑", Signature: "(x ∑) → ünïcode"},
	}
}

func nodesControl() []*schema.Node {
	return []*schema.Node{
		{Name: "Evil\x1b[31mRed\nName", Kind: "function", FilePath: "pkge/e.go", StartLine: 1, QualifiedName: "pkge.Evil\x1b[31mRed\nName", Signature: "func()\x1b[0m"},
	}
}

// ---------------------------------------------------------------------
// Plain-expectation helpers — copied verbatim from the RunE formats
// (search.go/callers.go/callees.go/impact.go/affected.go). These are the
// strip target: NEVER built from a lipgloss render.
// ---------------------------------------------------------------------

func sanitizeControlForTest(s string) string {
	// Mirrors sanitizeControl (sanitize.go) exactly — used only to build
	// the plain expectation for the control-byte fixture; production code
	// path is sanitizeControl itself, exercised via the renderer.
	return sanitizeControl(s)
}

func plainLocationLine(indent string, l query.Location) string {
	return fmt.Sprintf("%s%s (%s) %s:%d\n", indent,
		sanitizeControlForTest(l.Name), sanitizeControlForTest(l.Kind),
		sanitizeControlForTest(l.FilePath), l.StartLine)
}

func plainSearch(locs []query.Location) string {
	var b strings.Builder
	for _, l := range locs {
		b.WriteString(plainLocationLine("", l))
	}
	return b.String()
}

func plainSearchFull(nodes []*schema.Node) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(plainLocationLine("", query.Location{Name: n.Name, Kind: n.Kind, FilePath: n.FilePath, StartLine: n.StartLine}))
		qn := sanitizeControlForTest(n.QualifiedName)
		sig := sanitizeControlForTest(n.Signature)
		if sig == "" {
			b.WriteString("    " + qn + "\n")
		} else {
			b.WriteString("    " + qn + "  " + sig + "\n")
		}
	}
	return b.String()
}

func plainCallers(symbol string, locs []query.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s has %d caller(s):\n", sanitizeControlForTest(symbol), len(locs))
	for _, l := range locs {
		b.WriteString(plainLocationLine("  ", l))
	}
	return b.String()
}

func plainCallees(symbol string, locs []query.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s calls %d callee(s):\n", sanitizeControlForTest(symbol), len(locs))
	for _, l := range locs {
		b.WriteString(plainLocationLine("  ", l))
	}
	return b.String()
}

func plainImpact(symbol string, depth, nodeCount, edgeCount int, locs []query.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s impact (depth=%d): %d node(s), %d edge(s)\n",
		sanitizeControlForTest(symbol), depth, nodeCount, edgeCount)
	for _, l := range locs {
		b.WriteString(plainLocationLine("  ", l))
	}
	return b.String()
}

func plainAffected(locs []query.Location) string {
	var b strings.Builder
	if len(locs) == 0 {
		b.WriteString("no test files affected\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%d affected test(s):\n", len(locs))
	for _, l := range locs {
		b.WriteString(plainLocationLine("  ", l))
	}
	return b.String()
}

// ---------------------------------------------------------------------
// The content contract (D-00): stripped-styled output == plain output,
// over 0/1/N/duplicate/unicode(+control) fixtures, for every renderer.
// ---------------------------------------------------------------------

func TestRenderResultsStrippedEqualsPlain(t *testing.T) {
	pal := NewPalette(true)

	t.Run("RenderSearch", func(t *testing.T) {
		cases := []struct {
			name string
			locs []query.Location
		}{
			{"empty", locsEmpty()},
			{"single", locsSingle()},
			{"three-unordered", locsThreeUnordered()},
			{"duplicate", locsDuplicate()},
			{"unicode", locsUnicode()},
			{"control-bytes", locsControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var buf bytes.Buffer
				if err := RenderSearch(c.locs, pal, &buf); err != nil {
					t.Fatalf("RenderSearch: %v", err)
				}
				want := plainSearch(c.locs)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderSearchFull", func(t *testing.T) {
		cases := []struct {
			name  string
			nodes []*schema.Node
		}{
			{"empty", nodesEmpty()},
			{"single", nodesSingle()},
			{"three-unordered", nodesThreeUnordered()},
			{"duplicate", nodesDuplicate()},
			{"unicode", nodesUnicode()},
			{"control-bytes", nodesControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				var buf bytes.Buffer
				if err := RenderSearchFull(c.nodes, pal, &buf); err != nil {
					t.Fatalf("RenderSearchFull: %v", err)
				}
				want := plainSearchFull(c.nodes)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderCallers", func(t *testing.T) {
		cases := []struct {
			name string
			locs []query.Location
		}{
			{"empty", locsEmpty()},
			{"single", locsSingle()},
			{"three-unordered", locsThreeUnordered()},
			{"duplicate", locsDuplicate()},
			{"unicode", locsUnicode()},
			{"control-bytes", locsControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				r := query.CallersResult{Symbol: "Target", Callers: c.locs}
				var buf bytes.Buffer
				if err := RenderCallers(r, pal, &buf); err != nil {
					t.Fatalf("RenderCallers: %v", err)
				}
				want := plainCallers(r.Symbol, c.locs)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderCallees", func(t *testing.T) {
		cases := []struct {
			name string
			locs []query.Location
		}{
			{"empty", locsEmpty()},
			{"single", locsSingle()},
			{"three-unordered", locsThreeUnordered()},
			{"duplicate", locsDuplicate()},
			{"unicode", locsUnicode()},
			{"control-bytes", locsControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				r := query.CalleesResult{Symbol: "Target", Callees: c.locs}
				var buf bytes.Buffer
				if err := RenderCallees(r, pal, &buf); err != nil {
					t.Fatalf("RenderCallees: %v", err)
				}
				want := plainCallees(r.Symbol, c.locs)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderImpact", func(t *testing.T) {
		cases := []struct {
			name string
			locs []query.Location
		}{
			{"empty", locsEmpty()},
			{"single", locsSingle()},
			{"three-unordered", locsThreeUnordered()},
			{"duplicate", locsDuplicate()},
			{"unicode", locsUnicode()},
			{"control-bytes", locsControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				r := query.ImpactResult{Symbol: "Target", Depth: 2, NodeCount: len(c.locs), EdgeCount: len(c.locs) * 2, Affected: c.locs}
				var buf bytes.Buffer
				if err := RenderImpact(r, pal, &buf); err != nil {
					t.Fatalf("RenderImpact: %v", err)
				}
				want := plainImpact(r.Symbol, r.Depth, r.NodeCount, r.EdgeCount, c.locs)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderAffected", func(t *testing.T) {
		cases := []struct {
			name string
			locs []query.Location
		}{
			{"empty", locsEmpty()},
			{"single", locsSingle()},
			{"three-unordered", locsThreeUnordered()},
			{"duplicate", locsDuplicate()},
			{"unicode", locsUnicode()},
			{"control-bytes", locsControl()},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				r := query.AffectedResult{AffectedTests: c.locs}
				var buf bytes.Buffer
				if err := RenderAffected(r, pal, &buf); err != nil {
					t.Fatalf("RenderAffected: %v", err)
				}
				want := plainAffected(c.locs)
				if got := stripANSI(buf.String()); got != want {
					t.Errorf("stripped output mismatch\n got: %q\nwant: %q", got, want)
				}
			})
		}
	})

	t.Run("RenderNotice", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			var buf bytes.Buffer
			if err := RenderNotice("", pal, &buf); err != nil {
				t.Fatalf("RenderNotice: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("expected no output for empty notice, got:\n%s", buf.String())
			}
		})
		t.Run("non-empty", func(t *testing.T) {
			notice := "⚠ worktree mismatch: index built from /a, current /b\n"
			var buf bytes.Buffer
			if err := RenderNotice(notice, pal, &buf); err != nil {
				t.Fatalf("RenderNotice: %v", err)
			}
			if got := stripANSI(buf.String()); got != notice {
				t.Errorf("stripped notice mismatch\n got: %q\nwant: %q", got, notice)
			}
		})
	})

	// Positive control (rule 84d1gfpywd): the STYLED N=3 output must
	// actually contain an SGR escape — a renderer that forgot to style
	// anything would otherwise pass the stripped-equals-plain contract
	// vacuously.
	t.Run("PositiveControl_StyledOutputContainsANSI", func(t *testing.T) {
		locs := locsThreeUnordered()
		var buf bytes.Buffer
		if err := RenderSearch(locs, pal, &buf); err != nil {
			t.Fatalf("RenderSearch: %v", err)
		}
		if !strings.Contains(buf.String(), "\x1b[") {
			t.Errorf("expected an ANSI escape sequence in styled output, got:\n%s", buf.String())
		}

		var callersBuf bytes.Buffer
		r := query.CallersResult{Symbol: "Target", Callers: locs}
		if err := RenderCallers(r, pal, &callersBuf); err != nil {
			t.Fatalf("RenderCallers: %v", err)
		}
		if !strings.Contains(callersBuf.String(), "\x1b[") {
			t.Errorf("expected an ANSI escape sequence in styled callers output, got:\n%s", callersBuf.String())
		}
	})
}
