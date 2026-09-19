package cli

import "github.com/spf13/cobra"

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "hook"}
	cmd.AddCommand(newHookPreToolUseCmd())
	return cmd
}

func newHookPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:  "pretooluse",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
}
