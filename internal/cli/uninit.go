package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newUninitCmd builds the `codegraph uninit` command: removes
// .codegraph/ cleanly (T-02-12). Requires interactive confirmation unless
// --force is passed (D-01a). os.RemoveAll is scoped strictly to the
// resolved .codegraph/ under the target root — never an arbitrary
// user-supplied path. If .codegraph/ is already absent, exits cleanly
// with a message rather than erroring.
func newUninitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninit [path]",
		Short: "Remove .codegraph/",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}

			codegraphDir := filepath.Join(root, codegraphDirName)
			if _, err := os.Stat(codegraphDir); os.IsNotExist(err) {
				fmt.Fprintf(cmd.OutOrStdout(), "%s does not exist — nothing to do\n", codegraphDir)
				return nil
			} else if err != nil {
				return err
			}

			if !force {
				ok, err := confirm(cmd, fmt.Sprintf("Remove %s?", codegraphDir))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "aborted (pass --force to remove without confirming)")
					return nil
				}
			}

			if err := os.RemoveAll(codegraphDir); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", codegraphDir)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "remove without prompting for confirmation")

	return cmd
}

// confirm prompts the user via cmd's configured input/output (SetIn/SetOut
// in tests, os.Stdin/os.Stdout in production) and reports whether they
// answered affirmatively ("y"/"yes", case-insensitive). Any other input —
// including EOF/no input — is treated as "no": destructive operations
// default to safe.
func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N]: ", prompt)
	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}
