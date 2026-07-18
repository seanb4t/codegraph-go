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
//
// 07-01 anchored bubbles/bubbletea here via blank imports so `go mod tidy`
// wouldn't prune the requires before a real importer existed; both
// agentpicker.go (07-06) and daemonpicker.go (07-07) now import them
// directly, so the anchor is no longer needed.
package tui
