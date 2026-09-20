package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// telemetryStatement is the honest, auditable trust claim `codegraph
// telemetry` prints (D-15/CLI-03): this build collects zero passive or
// background telemetry, and `codegraph upgrade` is disclosed as the one
// intentional, user-initiated exception — never "this binary never
// touches the network" (that would be dishonest; upgrade genuinely does).
// Wording is deliberately non-marketing so it reads as a verifiable claim,
// not a promise: readers are pointed at the SBOM and source to check it
// themselves.
const telemetryStatement = `This build of codegraph collects zero telemetry: there is no passive
or background phone-home code anywhere in this binary.

The one intentional exception is "codegraph upgrade": an explicit,
user-initiated network request to the project's GitHub Releases,
made only when you run that command. No other command opens a
network connection.

Verify this yourself: the SBOM lists every dependency shipped in
this binary, and the source is public — audit it for outbound
connections outside the upgrade command's package.`

// newTelemetryCmd builds the `codegraph telemetry` command: a static,
// auditable statement about the binary's network behavior (D-15,
// CLI-03). RunE performs no I/O beyond printing telemetryStatement — no
// network package is imported by this file (T-06-05-01's mitigation).
func newTelemetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "telemetry",
		Short:   "Print this build's telemetry/network-behavior statement",
		Long:    "Print a static, auditable statement about codegraph's telemetry and network behavior: zero passive collection, with `codegraph upgrade` disclosed as the sole user-initiated exception.",
		Example: "  codegraph telemetry",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := resolveColor(cmd)
			if mode.Styled {
				w := mode.Writer(cmd.OutOrStdout())
				pal := present.NewPalette(mode.Dark)
				// telemetryStatement carries no trailing newline (the
				// literal ends at "package."), so splitting on "\n" and
				// rendering each line via Lines — which appends its own
				// "\n" per call — reconstructs exactly the same bytes
				// fmt.Fprintln's single trailing newline produces on the
				// plain path.
				return present.Lines(w, pal, present.RoleValue, strings.Split(telemetryStatement, "\n")...)
			}

			fmt.Fprintln(cmd.OutOrStdout(), telemetryStatement)
			return nil
		},
	}
}
