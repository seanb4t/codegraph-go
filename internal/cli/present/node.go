package present

import (
	"io"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// nodeMultiDefHardCap, nodeMultiDefBodyBudget and nodeMultiDefListCap
// will duplicate internal/query/render_markdown.go's unexported NODE-02
// budget constants verbatim (GREEN phase) — declared here now so the
// RED-phase test file (which references them directly) compiles.
const (
	nodeMultiDefHardCap    = 16
	nodeMultiDefBodyBudget = 12000
	nodeMultiDefListCap    = 20
)

// RenderNode is implemented in the GREEN phase of this TDD plan (04-05,
// Task 1). This placeholder deliberately writes nothing so the RED-phase
// contract tests fail on their content assertion, never on a compile or
// crash (#3770).
func RenderNode(d query.NodeDetail, pal Palette, w io.Writer) error {
	return nil
}
