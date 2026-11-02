package swiftextract

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestLabelsDistinguishFunctions checks design.md D16 for Swift: the
// qualified name carries the argument labels, so `move(to:)` and
// `move(by:)` are two functions with two ids, and a call records the
// labels it passes.
func TestLabelsDistinguishFunctions(t *testing.T) {
	src := `func move(to point: Int) {}
func move(by delta: Int) {}
func run() {
    move(by: 1)
}
`
	p := newTestParser(t)
	res, err := Extract(p, "app", "app.swift", []byte(src))
	if err != nil || res.Err != nil {
		t.Fatalf("Extract: %v / %v", err, res.Err)
	}
	got := map[string]string{}
	for _, en := range res.Nodes {
		if en.Node.Kind == goextract.KindFunction && en.Node.Name == "move" {
			got[en.Node.QualifiedName] = en.Node.Id
		}
	}
	if len(got) != 2 || got["move(to:)"] == "" || got["move(by:)"] == "" || got["move(to:)"] == got["move(by:)"] {
		t.Fatalf("move functions = %v, want move(to:) and move(by:) with distinct ids", got)
	}
	for _, r := range res.Unresolved {
		if r.Kind == goextract.RefKindCalls && r.Name == "move" {
			if len(r.ArgLabels) != 1 || r.ArgLabels[0] != "by" {
				t.Errorf("call labels = %v, want [by]", r.ArgLabels)
			}
			return
		}
	}
	t.Error("no recorded call to move")
}
