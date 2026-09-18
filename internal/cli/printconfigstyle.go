package cli

import (
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// configStyleFields builds one target's --print-config-style rendering at
// loc (D-04): "scopes=<csv> mcp=<p> format=<f> instructions=<p|none>
// skill=<dir|none> hooks=<h>", or "scopes=<csv> (<loc> not supported)"
// when t does not support loc.
//
// RED placeholder — returns the zero value; GREEN implements the real
// body per the plan's <action> block.
func configStyleFields(t agents.AgentTarget, loc agents.Location) (string, error) {
	return "", nil
}

// printConfigStyle renders every target in targets at loc, one line per
// target, in the order given — the read-only body of `install
// --print-config-style` (D-04). Zero targets prints "no agents selected".
//
// RED placeholder — writes nothing and returns nil; GREEN implements the
// real body per the plan's <action> block.
func printConfigStyle(cmd *cobra.Command, targets []agents.AgentTarget, loc agents.Location) error {
	return nil
}
