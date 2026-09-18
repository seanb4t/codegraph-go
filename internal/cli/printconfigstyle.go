package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// scopesCSV renders caps' declared Scopes as a comma-separated list in the
// fixed order global,local, filtered to membership — the order
// --print-config-style always renders in regardless of a target's own
// literal ordering (AGENT-08 ordering).
func scopesCSV(caps agents.Capabilities) string {
	var parts []string
	for _, loc := range []agents.Location{agents.LocationGlobal, agents.LocationLocal} {
		if caps.Supports(loc) {
			parts = append(parts, string(loc))
		}
	}
	return strings.Join(parts, ",")
}

// configStyleFields builds one target's --print-config-style rendering at
// loc (D-04): "scopes=<csv> mcp=<p> format=<f> instructions=<p|none>
// skill=<dir|none> hooks=<h>", or "scopes=<csv> (<loc> not supported)"
// when t does not support loc. Every resolved path is sanitized before
// it reaches any renderer (T-05-02).
func configStyleFields(t agents.AgentTarget, loc agents.Location) (string, error) {
	caps := t.Capabilities()
	scopes := scopesCSV(caps)

	if !caps.Supports(loc) {
		return fmt.Sprintf("scopes=%s (%s not supported)", scopes, loc), nil
	}

	if caps.MCPConfig == nil {
		return "", fmt.Errorf("%s: resolve mcp path: no MCPConfig declared for a supported location", t.ID())
	}
	mcp, err := caps.MCPConfig(loc)
	if err != nil {
		return "", fmt.Errorf("%s: resolve mcp path: %w", t.ID(), err)
	}

	instr, err := caps.InstructionsPath(loc)
	if err != nil {
		return "", fmt.Errorf("%s: resolve instructions path: %w", t.ID(), err)
	}
	instrDisplay := "none"
	if instr != "" {
		instrDisplay = sanitizePathForDisplay(instr)
	}

	skill, err := caps.WrittenSkillDir(loc)
	if err != nil {
		return "", fmt.Errorf("%s: resolve skill path: %w", t.ID(), err)
	}
	skillDisplay := "none"
	if skill != "" {
		skillDisplay = sanitizePathForDisplay(skill)
	}

	return fmt.Sprintf("scopes=%s mcp=%s format=%s instructions=%s skill=%s hooks=%s",
		scopes, sanitizePathForDisplay(mcp), caps.ConfigFormat, instrDisplay, skillDisplay, caps.Hooks), nil
}

// printConfigStyle renders every target in targets at loc, one line per
// target, in the order given — the read-only body of `install
// --print-config-style` (D-04). Zero targets prints "no agents selected".
// Plain rendering only; Task 3 adds the styled branch behind
// resolveColor(cmd).
func printConfigStyle(cmd *cobra.Command, targets []agents.AgentTarget, loc agents.Location) error {
	out := cmd.OutOrStdout()

	if len(targets) == 0 {
		fmt.Fprintln(out, "no agents selected")
		return nil
	}

	for _, t := range targets {
		fields, err := configStyleFields(t, loc)
		if err != nil {
			return fmt.Errorf("codegraph install: %w", err)
		}
		fmt.Fprintf(out, "%s: %s\n", t.ID(), fields)
	}
	return nil
}
