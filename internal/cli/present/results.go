package present

import (
	"io"

	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Placeholder (RED phase, 04-04): every renderer below is unimplemented
// and intentionally writes nothing so results_test.go's
// TestRenderResultsStrippedEqualsPlain fails on the byte comparison. The
// GREEN commit replaces these bodies with the real implementations.

// RenderSearch will render locs — the search default shape.
func RenderSearch(locs []query.Location, pal Palette, w io.Writer) error {
	return nil
}

// RenderSearchFull will render nodes — the search --full shape.
func RenderSearchFull(nodes []*schema.Node, pal Palette, w io.Writer) error {
	return nil
}

// RenderCallers will render r — the callers shape.
func RenderCallers(r query.CallersResult, pal Palette, w io.Writer) error {
	return nil
}

// RenderCallees will render r — the callees shape.
func RenderCallees(r query.CalleesResult, pal Palette, w io.Writer) error {
	return nil
}

// RenderImpact will render r — the impact shape.
func RenderImpact(r query.ImpactResult, pal Palette, w io.Writer) error {
	return nil
}

// RenderAffected will render r — the affected shape.
func RenderAffected(r query.AffectedResult, pal Palette, w io.Writer) error {
	return nil
}

// RenderNotice will render notice — the WORK-02 worktree notice.
func RenderNotice(notice string, pal Palette, w io.Writer) error {
	return nil
}
