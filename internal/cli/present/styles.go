// Package present is the sole home for charm.land/lipgloss/v2 styling
// (D-01). It consumes already-computed plain data (query.StatusResult,
// the files result struct) and emits colorized, sectioned output;
// internal/query and internal/mcp never import charm — the boundary the
// TUI-01 archtest (internal/cli/present/archtest) enforces at build time.
//
// present must NOT read os.Getenv or call term.IsTerminal itself — real
// fd/env values are read only at the RunE call sites in internal/cli
// (D-03).
package present

import lipgloss "charm.land/lipgloss/v2"

// Shared lipgloss style palette used by present's renderers. lipgloss v2
// removed the renderer-construction API present in v1 — Style.Render()
// always emits full-fidelity ANSI; downsampling is an explicit, separate
// concern this package does not need since the plain branch already
// bypasses lipgloss entirely on a non-TTY (D-04).
var (
	// headerStyle renders section titles (e.g. "CodeGraph Status").
	headerStyle = lipgloss.NewStyle().Bold(true)

	// labelStyle renders a stat/field key (e.g. "Files", "Nodes").
	labelStyle = lipgloss.NewStyle().Faint(true)

	// sectionStyle renders a section heading (e.g. "Index Statistics:").
	sectionStyle = lipgloss.NewStyle().Bold(true).Underline(true)
)
