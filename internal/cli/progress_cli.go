package cli

import (
	"os"

	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// startProgress wires the TTY-gated stderr spinner shared by init/index/sync
// (TUI-05/D-07/D-08, WR-04 06-REVIEW.md — de-duplicates what was previously
// three verbatim copies of this block). Evaluated against os.Stderr's fd
// since progress writes to stderr, never stdout; --quiet also suppresses
// the spinner (existing --quiet contract). Returns a cleanup func the
// caller must defer unconditionally — a no-op when the pretty branch
// didn't fire, or Progress.Stop (idempotent) when it did.
func startProgress(quiet bool, label string) func() {
	if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
		return func() {}
	}
	prog := present.NewProgress(os.Stderr)
	prog.Start(label)
	return prog.Stop
}
