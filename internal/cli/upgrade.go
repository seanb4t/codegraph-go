package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/agents"
	"github.com/seanb4t/codegraph-go/internal/upgrade"
	"github.com/seanb4t/codegraph-go/internal/version"
)

// upgradeRunFunc matches upgrade.Run's signature. A package-level var
// (rather than a direct call) so upgrade_test.go can substitute a fake and
// assert the command's flag/arg wiring without ever touching the network —
// mirrors internal/upgrade's own injectable-seam pattern one level up.
var upgradeRunFunc = upgrade.Run

// refreshInstalledSkillsFunc is a package-level injectable seam over
// refreshInstalledSkills, following the same idiom upgradeRunFunc and
// install.go's interactiveAllowed/runAgentPicker already establish in this
// package, so upgrade_test.go can assert the refresh call without touching
// the real filesystem.
var refreshInstalledSkillsFunc = refreshInstalledSkills

// refreshInstalledSkills implements D-06: after a successful binary swap,
// re-invoke Install() for every Claude location that already carries a
// codegraph manifest, using the newly swapped binary at execPath. The
// manifest's presence (agents.ConfiguredSkillLocations) is the only record
// of what the user previously consented to configure — refresh touches
// exactly those locations and never a location or target the user had not
// already installed to (this plan's must_haves.prohibitions). When no
// location carries a manifest, this is a silent no-op: an upgrade on a
// machine that never ran `install` must stay silent, not print a
// confusing no-op line.
//
// AutoAllow is always passed as false. --auto-allow is a per-invocation
// choice the user made at install time and is not recorded in the
// manifest, so re-asserting it on every upgrade would silently re-add a
// permission the user may have deliberately removed afterward. Passing
// false is not equivalent to removing it: Install with AutoAllow: false
// simply skips the permission-list step rather than deleting anything
// already present, which is the correct neutral behavior here.
func refreshInstalledSkills(execPath string, out io.Writer) error {
	locs := agents.ConfiguredSkillLocations(agents.Claude)
	if len(locs) == 0 {
		return nil
	}

	var errs []error
	for _, loc := range locs {
		targets, err := agents.ResolveTargetFlag(string(agents.Claude), loc)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, t := range targets {
			result := t.Install(loc, agents.InstallOptions{ExecPath: execPath, AutoAllow: false})
			for _, f := range result.Files {
				fmt.Fprintf(out, "  %s: %s\n", f.Action, f.Path)
			}
			for _, note := range result.Notes {
				fmt.Fprintf(out, "  note: %s\n", note)
			}
			errs = append(errs, result.Errors...)
		}
	}
	return errors.Join(errs...)
}

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
			"a newer release is available without downloading anything.\n\n" +
			"A Homebrew-managed install is detected from the resolved location of\n" +
			"the running binary. codegraph upgrade refuses to run there and exits\n" +
			"non-zero, because it was asked for a mutation it declines to perform.\n" +
			"codegraph upgrade --check steps aside with the same pointer and exits\n" +
			"zero, because it only answered a question. Upgrade a Homebrew-managed\n" +
			"install with: brew upgrade codegraph.",
		Example: "  codegraph upgrade --check\n  codegraph upgrade\n  codegraph upgrade v1.4.0\n" +
			"  codegraph upgrade --check  # brew-managed install: prints the pointer, exits 0",
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

			swapErr := upgradeRunFunc(version.Info().Version, target, upgrade.Options{
				Check:   check,
				Version: pinned,
				Force:   force,
				Out:     cmd.OutOrStdout(),
			})
			if swapErr != nil {
				// A failed swap means the binary did not change — there
				// is nothing to refresh, and swapErr surfaces unchanged.
				return swapErr
			}
			if check {
				// --check answered a question and mutated nothing; the
				// refresh step must mutate nothing too.
				return nil
			}

			if refreshErr := refreshInstalledSkillsFunc(target, cmd.OutOrStdout()); refreshErr != nil {
				// D-07: the swap already succeeded and is independently
				// verified/atomic — a refresh failure is reported as a
				// separate warning, not as a failed upgrade. Conflating a
				// config-file write hiccup with "your upgrade didn't work"
				// would be actively misleading when the binary genuinely
				// did update, and would likely send the user to re-run
				// upgrade rather than the one command that actually fixes
				// it, so the warning names that command explicitly.
				fmt.Fprintf(cmd.OutOrStdout(), "warning: codegraph upgrade succeeded, but refreshing the installed agent skill package failed: %v\nRun `codegraph install` to refresh it manually.\n", refreshErr)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&check, "check", false, "report whether a newer release is available, without downloading")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "reinstall even if already on the latest version")

	return cmd
}
