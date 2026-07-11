// Package cli implements the codegraph command-line interface: the Cobra
// root command "codegraph" and its init/index/uninit subcommands, wired
// directly to the internal/indexer pipeline (D-01). Commands contain no
// extraction/resolution logic of their own — they resolve paths, manage
// the .codegraph/ directory layout (D-01b), and delegate all indexing work
// to indexer.Run. sync/daemon/unlock (D-01/D-05) extend the surface with
// incremental update, the shared watch/index server, and stale-lock
// clearing, delegating to indexer.Sync and internal/daemon respectively.
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

// newRootCmd builds the "codegraph" root command and attaches the
// init/index/uninit subcommands (D-01), plus sync/daemon/unlock (D-01/D-05)
// — the incremental-update surface: sync updates the graph in one shot,
// daemon runs the shared watch/index server, and unlock clears a stale
// daemon lock. Usage/error text is printed by the caller
// (cmd/codegraph/main.go), not by cobra itself, so SilenceUsage and
// SilenceErrors are set on every command in the tree.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "codegraph",
		Short: "Pre-indexed code knowledge graph for coding agents",
		Long: "codegraph builds and maintains a local knowledge graph of a " +
			"repository's symbols, edges, and files so coding agents can " +
			"query code structure without repeated grep/read passes.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
		newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
		newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
		newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd())
	return root
}

// Execute runs the codegraph root command against os.Args, returning any
// error for the caller (cmd/codegraph/main.go) to report and exit non-zero
// on.
func Execute() error {
	return newRootCmd().Execute()
}
