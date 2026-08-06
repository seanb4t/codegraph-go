// This file is the CI entrypoint for the D-03 anti-regeneration guard
// (see classify.go's package doc comment for the rule itself). It reads a
// merge-base `git diff --name-status` changed-file list and a merge-base
// go.mod diff, both as files, classifies them via Classify, and exits
// accordingly:
//
//	0 - clean: no violation, including when the changed-file list was
//	    read successfully and contained zero records — a pull request
//	    that touches nothing is clean, not broken. If Classify's
//	    SwapExemption fired (SDK-01's one-time mark3labs-to-go-sdk
//	    transition), the exemption notice is also printed to stderr before
//	    returning 0 — an exempted run is never a silent pass.
//	1 - Classify reported a violation; the verdict's Reason is printed to
//	    stderr.
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
	"os"
)

func main() {
	os.Exit(run())
}

func run() int {
	changedListPath := flag.String("changed-list", "", "path to a file holding the merge-base `git diff --name-status` output")
	gomodDiffPath := flag.String("gomod-diff", "", "path to a file holding the merge-base diff of go.mod (may point to an empty file)")
	flag.Parse()

	if *changedListPath == "" || *gomodDiffPath == "" {
		fmt.Fprintln(os.Stderr, "transcriptfreeze: both -changed-list and -gomod-diff are required")
		return 2
	}

	changedRaw, err := os.ReadFile(*changedListPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "transcriptfreeze: %v (-changed-list %s): %v\n", ErrNoInput, *changedListPath, err)
		return 2
	}
	gomodRaw, err := os.ReadFile(*gomodDiffPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "transcriptfreeze: %v (-gomod-diff %s): %v\n", ErrNoInput, *gomodDiffPath, err)
		return 2
	}

	changed, err := ParseChangedList(string(changedRaw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "transcriptfreeze: unusable changed-file list: %v\n", err)
		return 2
	}

	verdict := Classify(changed, string(gomodRaw))
	if verdict.SwapExemption {
		fmt.Fprintln(os.Stderr, verdict.Reason)
		return 0
	}
	if verdict.Violation {
		fmt.Fprintln(os.Stderr, verdict.Reason)
		return 1
	}
	return 0
}
