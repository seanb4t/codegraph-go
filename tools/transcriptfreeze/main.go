// This file is the CI entrypoint for the D-03 anti-regeneration guard
// (see classify.go's package doc comment for the rule itself). It reads a
// merge-base `git diff --name-status` changed-file list and a merge-base
// go.mod diff, both as files, classifies them via Classify, and exits
// accordingly:
//
//	0 - advisory: the guard never fails the build. This covers three
//	    distinct outcomes, distinguishable only via stderr, since 03-02
//	    made a detected collision advisory rather than blocking (the
//	    guard's own "split into two pull requests" remedy is
//	    structurally unavailable in this repository — see classify.go's
//	    package doc comment):
//	      - a genuinely clean diff: nothing is printed.
//	      - a detected D-03 collision: the full Verdict.Reason — naming
//	        both offending sides — is printed to stderr, but the run
//	        still exits 0. The report is the deliverable now that the
//	        exit code no longer is.
//	      - an SDK-01 swap exemption firing: the exemption notice is
//	        printed to stderr, exactly as before this change. An
//	        exempted run is never a silent pass.
//	2 - unusable input: either input file is missing or unreadable
//	    (wrapping ErrNoInput), or ParseChangedList returned a parse
//	    error. An unusable input must never be reported as a pass; an
//	    empty-but-readable input must never be reported as a failure —
//	    that distinction is the whole reason this file, not
//	    ParseChangedList, owns the file-read step.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run())
}

func run() int {
	return runCLI(os.Args[1:], os.Stderr)
}

// runCLI is run()'s testable core: it parses args with a fresh FlagSet
// (rather than the flag package's global CommandLine/os.Args) so tests can
// invoke it repeatedly with distinct inputs, reads the two input files,
// classifies them, and writes any report to stderr. This is the narrowest
// seam needed to give run() direct test coverage — run() itself has none
// today.
func runCLI(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("transcriptfreeze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	changedListPath := fs.String("changed-list", "", "path to a file holding the merge-base `git diff --name-status` output")
	gomodDiffPath := fs.String("gomod-diff", "", "path to a file holding the merge-base diff of go.mod (may point to an empty file)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *changedListPath == "" || *gomodDiffPath == "" {
		fmt.Fprintln(stderr, "transcriptfreeze: both -changed-list and -gomod-diff are required")
		return 2
	}

	changedRaw, err := os.ReadFile(*changedListPath)
	if err != nil {
		fmt.Fprintf(stderr, "transcriptfreeze: %v (-changed-list %s): %v\n", ErrNoInput, *changedListPath, err)
		return 2
	}
	gomodRaw, err := os.ReadFile(*gomodDiffPath)
	if err != nil {
		fmt.Fprintf(stderr, "transcriptfreeze: %v (-gomod-diff %s): %v\n", ErrNoInput, *gomodDiffPath, err)
		return 2
	}

	changed, err := ParseChangedList(string(changedRaw))
	if err != nil {
		fmt.Fprintf(stderr, "transcriptfreeze: unusable changed-file list: %v\n", err)
		return 2
	}

	verdict := Classify(changed, string(gomodRaw))
	if verdict.SwapExemption {
		// An exemption firing is never a silent pass.
		fmt.Fprintln(stderr, verdict.Reason)
		return 0
	}
	if verdict.Violation {
		// Advisory since 03-02 (option-advisory): the guard's own
		// remedy — split into two pull requests — is structurally
		// unavailable in this repository (03-CONTEXT.md D-06), so a
		// detected collision is reported in full but no longer blocks
		// the build. The reviewed-diff pass D-06 mandates is the
		// control that applies now.
		fmt.Fprintln(stderr, verdict.Reason)
		return 0
	}
	return 0
}
