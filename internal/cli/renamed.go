package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newQueryCmd and newUnlockCmd build hidden rename stubs for the two verbs
// this phase folds away (Phase 3 VERB-03/VERB-04): `codegraph query <term>`
// into `codegraph search --full <term>`, and `codegraph unlock [path]` into
// `codegraph daemon unlock [path]`. Both constructor names are unchanged
// from the commands they replace — root.go's AddCommand list still calls
// newQueryCmd()/newUnlockCmd() and needed no registration edit.
//
// Each stub's RunE does exactly two things: write the two-line rename
// notice (D-06, verbatim, em dash U+2014 included) to cmd.ErrOrStderr(),
// and return a non-nil error. Nothing else — no index or lock store is
// ever touched, and no arg/flag is inspected or forwarded to the new verb.
//
// Both fields below are set to true on the returned command (VERB-06,
// mirroring man.go's Hidden precedent for the first one):
//
//   - Hiding the command takes the stub out of the generated reference,
//     shell completions, and man-page tree — tools/clidoc,
//     cli_reference_test.go's documentedByReference, and cobra's own
//     __complete/doc.GenManTree machinery all skip a hidden command. It
//     still registers cobra's default --help flag (InitDefaultHelpFlag),
//     which is why each stub carries its own command-level entry in
//     testdata/cli-reference-allowlist.txt (D-12) — a bare command-path
//     entry covers exactly a hidden command's flags.
//   - Disabling flag parsing (Pitfall 2, 03-RESEARCH.md) is required so an
//     invocation like `query --json main` reaches RunE at all: leaving
//     ordinary flag parsing on would make pflag reject an unrecognized
//     flag before RunE ever runs, and the caller would see cobra's own
//     "unknown flag" error instead of the D-06 rename message. With it
//     disabled, cobra performs NO flag parsing at all for this command —
//     args (including anything that looks like a flag) arrive in RunE's
//     args slice unexamined, which is exactly right here since nothing in
//     the stub ever reads args.
//
// Cobra's deprecation field is deliberately left unset (D-05): setting it
// only adds a warning line while still EXECUTING the command — the
// opposite of what a rename stub needs, which is to run nothing.
//
// The stub calls no process-exit function itself. Returning a non-nil
// error is what makes cmd/codegraph/main.go's single error-reporting exit
// path — the tree's only error exit — fire; no new exit code is
// introduced.
//
// Both `query` and `unlock` are removed entirely in v0.15.0 (the next
// minor release, D-09) — tracked here, in the D-06 stderr text itself,
// in the two allowlist entries (D-12), and in the feat! commit's
// BREAKING CHANGE footer.
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "query",
		Short:              `Renamed to "search --full" — stub removed in v0.15.0`,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), `"query" has been renamed to "search --full" — run: codegraph search --full <term>`)
			fmt.Fprintln(cmd.ErrOrStderr(), `the "query" stub is removed in the next minor release (v0.15.0)`)
			return fmt.Errorf(`codegraph: "query" has been renamed to "search --full"`)
		},
	}
}

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "unlock",
		Short:              `Renamed to "daemon unlock" — stub removed in v0.15.0`,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), `"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]`)
			fmt.Fprintln(cmd.ErrOrStderr(), `the "unlock" stub is removed in the next minor release (v0.15.0)`)
			return fmt.Errorf(`codegraph: "unlock" has been renamed to "daemon unlock"`)
		},
	}
}
