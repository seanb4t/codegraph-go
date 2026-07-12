package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/version"
)

// newVersionCmd builds the `codegraph version` command: prints the
// ldflags-injected build identity (semver, git commit, build date) plus
// the runtime Go version and target os/arch (D-09). --json emits the
// same fields as machine-readable JSON, consumed by `upgrade --check`'s
// version comparison (06-06).
func newVersionCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Print build version information",
		Long:    "Print the codegraph binary's build identity: semver, git commit, build date, Go version, and os/arch.",
		Example: "  codegraph version\n  codegraph version --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Info()
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(info)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "codegraph %s (commit %s, built %s) %s %s/%s\n",
				info.Version, info.Commit, info.Date, info.GoVersion, info.OS, info.Arch)
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "emit build identity as JSON")

	return cmd
}

// versionLine formats a one-line version string for root.Version, so
// `codegraph --version` (Cobra's built-in flag) prints the same build
// identity as `codegraph version`.
func versionLine() string {
	info := version.Info()
	return fmt.Sprintf("%s (commit %s, built %s)", info.Version, info.Commit, info.Date)
}
