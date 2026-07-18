// Package tui hosts the ONLY code in this module that imports
// charm.land/bubbletea/v2 and charm.land/bubbles/v2 (D-09/D-11). It is a
// sibling of internal/cli/present, not a subpackage of it, so the pure,
// side-effect-free TTY-gate helpers in present stay untouched while this
// package grows the stateful, interactive components (daemon picker,
// install/uninstall multi-select) that later plans in this phase add.
//
// internal/cli/tui is excluded from the TUI-01 archtest's guarded closure
// (internal/cli/present/archtest/import_graph_test.go already excludes the
// whole internal/cli prefix), so importing bubbletea/bubbles here can never
// leak into the serve-reachable surface (internal/daemon, internal/mcp,
// etc.) by construction.
package tui

// Anchor imports: 07-01 adds the two interactive dependencies to go.mod
// ahead of the bubbletea Models that use them for real (07-06's daemon
// picker, 07-07's install/uninstall multi-select). Without a real importer
// yet, `go mod tidy` would otherwise prune these requires as unused. Remove
// this file's blank imports once daemonpicker.go/agentpicker.go land and
// import these packages directly.
import (
	_ "charm.land/bubbles/v2"
	_ "charm.land/bubbletea/v2"
)
