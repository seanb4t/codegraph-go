package cli

import "github.com/spf13/cobra"

// addSignatureFlag registers the --signature flag on cmd, bound to
// signature. node, callers, callees and impact call it, so the name and
// the help text have one place.
func addSignatureFlag(cmd *cobra.Command, signature *string) {
	cmd.Flags().StringVar(signature, "signature", "", "parameter types that select one overload, for example \"String, Throwable\"")
}

// addFileFlag registers the --file flag of callers, callees and impact
// on cmd, bound to file. It is a hard filter there. node has its own
// --file, which also selects the file mode of that command.
func addFileFlag(cmd *cobra.Command, file *string) {
	cmd.Flags().StringVarP(file, "file", "f", "", "file path — keeps the definitions of the symbol in the files whose path contains it")
}
