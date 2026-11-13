package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)

// addSignatureFlag registers the --signature flag on cmd, bound to
// signature. node, callers, callees and impact call it, so the name and
// the help text have one place.
func addSignatureFlag(cmd *cobra.Command, signature *string) {
	cmd.Flags().StringVar(signature, "signature", "", "parameter types that select one overload, for example \"String, Throwable\"")
}

// addFileFlag registers --file (short form -f) on cmd, bound to file,
// with the help text usage. node, callers, callees and impact call it.
// The meaning differs, so each command gives its own text: callers,
// callees and impact pass fileFilterUsage, a hard filter, and node
// passes a text that says the flag also selects its file mode.
func addFileFlag(cmd *cobra.Command, file *string, usage string) {
	cmd.Flags().StringVarP(file, "file", "f", "", usage)
}

// fileFilterUsage is the --file help text of callers, callees and impact.
const fileFilterUsage = "file path — keeps the definitions of the symbol in the files whose path contains it"

// printDefinitionsLine prints the definitions line of a query that spans
// definitions, in the plain output of callers, callees and impact. It
// prints nothing for a query over one definition or one overload set.
func printDefinitionsLine(out io.Writer, symbol string, defs []query.Location) {
	if line := query.DefinitionsLine(symbol, defs); line != "" {
		fmt.Fprintln(out, line)
	}
}
