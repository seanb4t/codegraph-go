// Package present is the sole home for charm.land/lipgloss/v2 styling
// (D-01). It consumes already-computed plain data (query.StatusResult,
// the files result struct) and emits colorized, sectioned output;
// internal/query and internal/mcp never import charm — the boundary the
// TUI-01 archtest (internal/cli/present/archtest) enforces at build time.
//
// present must NOT read the process environment or probe terminal state
// itself — real fd/env values are read only at the RunE call sites in
// internal/cli (D-03).
//
// The shared style palette used to live here as three unexported
// package-level lipgloss.Style variables. It now folds into palette.go's
// Palette, built per call by NewPalette(dark bool) from the resolver's
// single background-detection answer (D-05) — no mutable package-level
// style, no setter. Section headings that used to render via a dedicated
// underlined variable now render via Palette.Header; there is no eighth
// role.
package present
