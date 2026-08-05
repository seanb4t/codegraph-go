// Command wireoracle is the human-redirect capture entrypoint (D-03): it
// spawns one scenario against a caller-supplied binary and fixture, prints
// ONLY the normalized transcript to stdout, and prints the per-rule hit
// ledger to stderr. There is no flag or environment variable that writes a
// transcript file in place — freezing a transcript is a deliberate,
// reviewable, human-run `> file` redirect, never an automated regeneration
// path.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/seanb4t/codegraph-go/test/wireoracle"
)

func main() {
	binFlag := flag.String("bin", "", "path to the codegraph binary to capture against")
	fixtureFlag := flag.String("fixture", "", "path to the dedicated wire-oracle fixture tree")
	scenarioFlag := flag.String("scenario", "", "scenario name to capture (see wireoracle.Scenarios)")
	flag.Parse()

	if *binFlag == "" || *fixtureFlag == "" || *scenarioFlag == "" {
		fmt.Fprintln(os.Stderr, "usage: wireoracle -bin <path> -fixture <path> -scenario <name>")
		os.Exit(2)
	}

	sc, ok := wireoracle.ScenarioByName(*scenarioFlag)
	if !ok {
		fmt.Fprintf(os.Stderr, "wireoracle: unknown scenario %q\n", *scenarioFlag)
		os.Exit(2)
	}

	workDir, err := os.MkdirTemp("", "wireoracle-capture-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	tr, err := wireoracle.Capture(context.Background(), *binFlag, *fixtureFlag, workDir, sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: capture: %v\n", err)
		os.Exit(1)
	}

	normalized, ledger := wireoracle.NormalizeWithLedger(tr.Stdout, wireoracle.Substitutions{RepoDir: tr.RepoDir})

	for _, rule := range wireoracle.Rules {
		fmt.Fprintf(os.Stderr, "rule=%s hits=%d\n", rule.Name, ledger[rule.Name])
	}

	if _, err := os.Stdout.Write(normalized); err != nil {
		fmt.Fprintf(os.Stderr, "wireoracle: write stdout: %v\n", err)
		os.Exit(1)
	}
}
