// Command mcpaudit is the entrypoint for the VRFY-05 proxying MCP capture
// shim. See proxy.go's package doc for the measurement contract this
// binary implements; this file owns only flag parsing and process exit
// codes.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is main's testable core: it never calls os.Exit itself, so tests
// can drive it with scripted stdio and inspect the returned code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcpaudit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	real := fs.String("real", "", "absolute path to the real codegraph binary (required)")
	logPath := fs.String("log", "", "observation log path (required)")
	argsFlag := fs.String("args", "serve,--mcp", "comma-separated args passed to -real")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Fail loudly and immediately on a missing required flag — a
	// silently-misconfigured shim would produce an empty log that reads
	// like "the client sent nothing", which is indistinguishable from a
	// genuine UNMEASURED result without this check.
	if *real == "" || *logPath == "" {
		fmt.Fprintln(stderr, "mcpaudit: -real and -log are both required")
		return 2
	}

	var realArgs []string
	for _, a := range strings.Split(*argsFlag, ",") {
		if a != "" {
			realArgs = append(realArgs, a)
		}
	}

	// A real agent client can terminate this shim process directly rather
	// than closing its stdin (observed live against Claude Code during
	// the Task 2 audit) — without catching the signal here, an
	// unhandled SIGTERM/SIGINT kills the process before Run's
	// post-exit appendObservation ever runs, silently dropping the
	// partial-observation guarantee. NotifyContext intercepts exactly
	// once; a second signal falls through to default (immediate) behavior.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg := Config{
		RealBin:  *real,
		RealArgs: realArgs,
		LogPath:  *logPath,
		In:       stdin,
		Out:      stdout,
		ErrOut:   stderr,
		Ctx:      ctx,
	}

	if err := Run(cfg); err != nil {
		var exitErr *ExitCodeError
		if errors.As(err, &exitErr) {
			return exitErr.Code
		}
		fmt.Fprintf(stderr, "mcpaudit: %v\n", err)
		return 1
	}
	return 0
}
