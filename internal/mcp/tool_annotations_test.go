package mcp

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestEveryToolIsAnnotatedReadOnlyClosedWorld pins the risk vocabulary
// every codegraph tool advertises. The values it asserts were wrong for the
// whole of two milestones — inherited mark3labs zero-values that rendered a
// graph query as "destructive, open-world" in a real client's tool list —
// and nothing compared them against what the handlers actually do.
//
// The tool set is derived from exploreTool()/companionTool() via
// companionNames, this package's own source of truth, rather than a re-typed
// list of eight names: a ninth tool added without annotations inherits
// go-sdk's own defaults (readOnlyHint false, openWorldHint true) and fails
// here on its first run, which is the specific regression this test exists
// to make loud.
func TestEveryToolIsAnnotatedReadOnlyClosedWorld(t *testing.T) {
	tools := []*mcp.Tool{exploreTool()}
	for _, name := range companionNames {
		tools = append(tools, companionTool(name))
	}

	if len(tools) != len(allToolNames()) {
		t.Fatalf("built %d tool descriptors but allToolNames() reports %d — the two derivations have drifted", len(tools), len(allToolNames()))
	}

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			a := tool.Annotations
			if a == nil {
				t.Fatalf("%s carries no annotations, so it inherits go-sdk's defaults (readOnlyHint false, destructiveHint true, openWorldHint true) — every one of which is wrong for a read-only local graph query", tool.Name)
			}

			// Every handler reaches the store only through openEngine,
			// which yields a graphstore.Reader — an interface with no
			// mutator. See toolAnnotations' per-tool audit.
			if !a.ReadOnlyHint {
				t.Errorf("%s: readOnlyHint = false, want true", tool.Name)
			}

			// destructiveHint is "meaningful only when readOnlyHint ==
			// false" per the spec. Setting it either way alongside
			// readOnlyHint:true publishes an incoherent pair.
			if a.DestructiveHint != nil {
				t.Errorf("%s: destructiveHint = %v, want it omitted — the spec defines it as meaningful only when readOnlyHint is false", tool.Name, *a.DestructiveHint)
			}

			// Local pre-built index, no network, only read-only `git
			// rev-parse` subprocesses for worktree detection.
			if a.OpenWorldHint == nil {
				t.Errorf("%s: openWorldHint omitted, which the spec defaults to TRUE — the exact wrong answer for a closed local index", tool.Name)
			} else if *a.OpenWorldHint {
				t.Errorf("%s: openWorldHint = true, want false", tool.Name)
			}
		})
	}
}
