package cli

import (
	"os"

	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// startProgress wires the TTY-gated stderr spinner shared by init/index/sync
// (TUI-05/D-07/D-08, WR-04 06-REVIEW.md — de-duplicates what was previously
// three verbatim copies of this block). Evaluated against os.Stderr's fd
// since progress writes to stderr, never stdout; --quiet also suppresses the
// spinner (existing --quiet contract). Returns a cleanup func the caller must
// defer unconditionally — a no-op when the pretty branch didn't fire, or the
// idempotent Progress.Stop when it did.
//
// No signal handling by design: indexer.Run/Sync take no context and cannot be
// cancelled mid-run, so intercepting SIGINT/SIGTERM here made the process
// UNINTERRUPTIBLE — the first Ctrl-C cleared the spinner but signal.Notify
// stayed registered while the (context-unaware) work ran on, swallowing every
// later Ctrl-C for the rest of the run (06-REVIEW iter-2 CR-01). Ctrl-C
// therefore keeps Go's default terminating disposition; a long index/sync stays
// interruptible. The only cost is a cosmetic stray spinner frame left on the
// terminal when the process is killed mid-run — WR-02, accepted as a minor
// known issue (far preferable to an uninterruptible index). A correct
// clear-then-terminate handler needs a context threaded into indexer.Run/Sync;
// deferred as a future enhancement.
func startProgress(quiet bool, label string) func() {
	if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
		return func() {}
	}
	prog := present.NewProgress(os.Stderr)
	prog.Start(label)
	return prog.Stop
}
