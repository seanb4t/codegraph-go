package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// newManCmd builds the hidden `codegraph man <dir>` command: it writes the
// full Cobra command-tree man page set (doc.GenManTree — codegraph.1 plus
// one page per registered subcommand) into an explicit output directory
// (D-01). It exists specifically so the
// Homebrew cask's post-install hook
// (.goreleaser.yaml homebrew_casks.hooks.post.install) can generate man
// pages by running the INSTALLED binary at install time, rather than
// shipping a committed or release-time-generated copy that could drift
// from the binary that actually runs (D-05) — the same "generated from
// the binary, not checked in" guarantee `generate_completions_from_executable`
// already gives shell completions (D-06).
//
// Hidden: true (D-02) — deliberately NOT documented as a public,
// interactive command the way `githooks` is. `man` exists purely as a
// mechanism the cask hook invokes; a human never needs to run it
// directly, so its divergence footprint stays documented in this comment
// rather than on a new public surface. This is the one place
// this command's shape diverges from the internal/cli/githooks.go
// precedent it otherwise mirrors (registration convention, doc-comment
// decision-id citation) — githooks is NOT hidden.
//
// RunE returns doc.GenManTree's error directly, wrapped with the target
// directory, rather than printing a friendly line and returning nil: a
// hidden command invoked by a Ruby postflight hook has no interactive
// user to address. What matters is the non-nil exit status Homebrew's
// system_command raises on (D-10), not stdout a human would read.
//
// Does not reuse the package-level targetRoot(args): that helper resolves
// the codegraph project root (the directory containing .codegraph/), an
// unrelated concept to man's arbitrary output directory argument (D-04).
//
// Creates the target directory (including missing parents) before
// generating: doc.GenManTree does not create its own destination, and the
// cask hook targets Homebrew's man1 directory, which is absent on a prefix
// where nothing has yet installed a man page.
func newManCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "man <dir>",
		Short:  "Generate man pages for the full command tree",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create man page directory %s: %w", dir, err)
			}
			header := &doc.GenManHeader{
				Title:   "CODEGRAPH",
				Section: "1",
			}
			if err := doc.GenManTree(newRootCmd(), header, dir); err != nil {
				return fmt.Errorf("generate man pages into %s: %w", dir, err)
			}
			return nil
		},
	}
}
