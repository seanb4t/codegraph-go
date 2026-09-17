// Package cli implements the codegraph command-line interface: the Cobra
// root command "codegraph" and its init/index/uninit subcommands, wired
// directly to the internal/indexer pipeline (D-01). Commands contain no
// extraction/resolution logic of their own — they resolve paths, manage
// the .codegraph/ directory layout (D-01b), and delegate all indexing work
// to indexer.Run. sync/daemon (D-01/D-05) extend the surface with
// incremental update and the shared watch/index server, delegating to
// indexer.Sync and internal/daemon respectively. search is the one
// lexical-search verb (Phase 3 VERB-01/VERB-03): its --full flag selects
// the full-node-record shape the now-removed `query` verb used to render,
// and daemon start|stop|unlock (Phase 3 VERB-04) is the complete daemon
// lifecycle, unlock included — `daemon unlock` clears a stale daemon lock
// left behind by a crash. `query` and `unlock` remain registered only as
// hidden rename stubs (renamed.go) that print the rename and exit 1,
// removed entirely in v0.15.0. githooks (phase 05) manages marker-fenced
// git sync hooks, and man (Phase 3 D-01/D-02) is the hidden man-page
// generator the Homebrew cask's post-install hook invokes — both are
// documented Go-only surface extensions. ui (Phase 1 SRV-01) runs a local,
// read-only Connect RPC server (internal/uiserver) over the repository's
// own index, foreground until Ctrl-C like serve/daemon start, sharing
// neither's lifecycle.
package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// ErrAlreadyInitialized is returned by `init` when .codegraph/ already
// exists at the target root (D-01a) — init never clobbers an existing
// store; the caller must run `codegraph index --force` to rebuild.
var ErrAlreadyInitialized = errors.New("cli: already initialized")

// ErrNotInitialized is returned by `index` when no .codegraph/ exists yet
// at the target root (D-01a) — the caller must run `codegraph init` first.
var ErrNotInitialized = errors.New("cli: not initialized")

// The four cobra command groups D-13 assigns every visible command to
// (CLI-06) — registered on root via AddGroup in this exact order (query,
// build, agents, maintenance) so both cobra's own help template and, when
// fang is declined (D-14), present.RenderHelp render the same four
// titled sections in the same order. Hidden commands (man, the query/
// unlock rename stubs) are deliberately absent from commandGroups below
// and stay groupless.
const (
	groupQuery       = "query"
	groupBuild       = "build"
	groupAgents      = "agents"
	groupMaintenance = "maintenance"
)

// commandGroups is the ONE table mapping a command's Name() to its D-13
// group — applied to root's direct children after AddCommand (CLI-06).
// Only root's direct children are grouped: cobra groups are per-parent,
// so daemon start|stop and githooks install|remove|status are never
// grouped. man, query and unlock are deliberately absent (hidden, stay
// groupless); help and completion are set separately via
// SetHelpCommandGroupID/SetCompletionCommandGroupID since cobra creates
// them lazily, after this map is applied.
var commandGroups = map[string]string{
	"explore":  groupQuery,
	"search":   groupQuery,
	"node":     groupQuery,
	"callers":  groupQuery,
	"callees":  groupQuery,
	"impact":   groupQuery,
	"affected": groupQuery,
	"files":    groupQuery,
	"status":   groupQuery,

	"init":     groupBuild,
	"index":    groupBuild,
	"sync":     groupBuild,
	"daemon":   groupBuild,
	"githooks": groupBuild,
	"uninit":   groupBuild,

	"serve":     groupAgents,
	"ui":        groupAgents,
	"install":   groupAgents,
	"uninstall": groupAgents,

	"version":   groupMaintenance,
	"upgrade":   groupMaintenance,
	"telemetry": groupMaintenance,
}

// newRootCmd builds the "codegraph" root command and attaches the
// init/index/uninit subcommands (D-01), plus sync/daemon (D-01/D-05) — the
// incremental-update surface: sync updates the graph in one shot, and
// daemon start|stop|unlock runs the shared watch/index server and manages
// its lifecycle, including clearing a stale lock (Phase 3 VERB-04, moved
// verbatim from the old top-level `unlock`) — and githooks (Phase 5), the
// marker-fenced git sync hook manager. search gains --full (Phase 3
// VERB-01/VERB-03), folding in the now-removed `query` verb's full-record
// shape; `query` and `unlock` stay registered as hidden rename stubs
// (renamed.go) for one release (D-05/D-09). man (Phase 3 D-01/D-02) is a
// hidden, Go-only command generating the full man-page tree from the
// binary itself, invoked by the Homebrew cask's post-install hook rather
// than by an interactive user. Usage/error text is printed by the
// caller (cmd/codegraph/main.go), not by cobra itself, so SilenceUsage and
// SilenceErrors are set on every command in the tree.
//
// The tree is grouped into D-13's four titled sections (CLI-06): Query
// the graph, Build the index, Agents & serving, and Maintenance — every
// visible command's GroupID is applied from the one commandGroups table
// above right after AddCommand, and help/completion are filed under
// Maintenance via SetHelpCommandGroupID/SetCompletionCommandGroupID.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "codegraph",
		Short:   "Pre-indexed code knowledge graph for coding agents",
		Version: versionLine(),
		Long: "codegraph builds and maintains a local knowledge graph of a " +
			"repository's symbols, edges, and files so coding agents can " +
			"query code structure without repeated grep/read passes.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	addColorFlag(root)
	root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
		newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
		newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
		newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
		newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
		newUpgradeCmd(), newInstallCmd(), newUninstallCmd(),
		newGithooksCmd(), newManCmd(), newUiCmd())

	root.AddGroup(
		&cobra.Group{ID: groupQuery, Title: "Query the graph:"},
		&cobra.Group{ID: groupBuild, Title: "Build the index:"},
		&cobra.Group{ID: groupAgents, Title: "Agents & serving:"},
		&cobra.Group{ID: groupMaintenance, Title: "Maintenance:"},
	)
	for _, c := range root.Commands() {
		if id, ok := commandGroups[c.Name()]; ok {
			c.GroupID = id
		}
	}
	root.SetHelpCommandGroupID(groupMaintenance)
	root.SetCompletionCommandGroupID(groupMaintenance)

	return root
}

// Execute runs the codegraph root command against os.Args, returning any
// error for the caller (cmd/codegraph/main.go) to report and exit non-zero
// on.
func Execute() error {
	return newRootCmd().Execute()
}

// NewRootCmd returns a freshly built root command tree, identical to what
// Execute() runs. It exists so tools/clidoc (DOCS-05's out-of-package CLI
// reference generator) can obtain the same command tree without this
// package importing cobra/doc or anything new; it changes no behavior of
// the shipped binary. newRootCmd itself is not renamed — every existing
// intra-package and cmd/codegraph call site keeps working untouched.
func NewRootCmd() *cobra.Command {
	return newRootCmd()
}
