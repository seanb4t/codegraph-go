package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/upgrade"
	"github.com/seanb4t/codegraph-go/internal/version"
)

// upgradeRunFunc matches upgrade.Run's signature. A package-level var
// (rather than a direct call) so upgrade_test.go can substitute a fake and
// assert the command's flag/arg wiring without ever touching the network —
// mirrors internal/upgrade's own injectable-seam pattern one level up.
var upgradeRunFunc = upgrade.Run

// newUpgradeCmd builds `codegraph upgrade [version] [--check]` (CLI-02,
// D-11): a thin command that resolves the running binary's path via
// os.Executable() (D-13 — the self-replace target) and the current
// version.Info().Version, then delegates the entire
// resolve→verify→swap orchestration to upgrade.Run. All security-critical
// logic (signature verification, atomic swap, fail-closed ordering) lives
// in internal/upgrade, not here.
func newUpgradeCmd() *cobra.Command {
	var check bool
	var force bool

	cmd := &cobra.Command{
		Use:   "upgrade [version]",
		Short: "Download, verify, and install a new codegraph release",
		Long: "Download the target-platform binary from GitHub Releases, verify its\n" +
			"cosign-keyless signature/provenance in-process (never a cosign CLI), and\n" +
			"only then atomically replace the running binary. --check reports whether\n" +
			"a newer release is available without downloading anything.",
		Example: "  codegraph upgrade --check\n  codegraph upgrade\n  codegraph upgrade v1.4.0",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pinned string
			if len(args) > 0 {
				pinned = args[0]
			}

			target, err := os.Executable()
			if err != nil {
				return fmt.Errorf("codegraph upgrade: resolve running binary path: %w", err)
			}

			return upgradeRunFunc(version.Info().Version, target, upgrade.Options{
				Check:   check,
				Version: pinned,
				Force:   force,
				Out:     cmd.OutOrStdout(),
			})
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "report whether a newer release is available, without downloading")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "reinstall even if already on the latest version")

	return cmd
}
