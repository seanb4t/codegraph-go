// Command codegraph is the codegraph CLI binary entrypoint. All behavior
// lives in internal/cli — main only delegates to it and translates a
// returned error into a non-zero exit code.
package main

import (
	"fmt"
	"os"

	"github.com/seanb4t/codegraph-go/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
