package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/seanb4t/codegraph-go/internal/cli/present"
)

// startProgress wires the TTY-gated stderr spinner shared by init/index/sync
// (TUI-05/D-07/D-08, WR-04 06-REVIEW.md — de-duplicates what was previously
// three verbatim copies of this block). Evaluated against os.Stderr's fd
// since progress writes to stderr, never stdout; --quiet also suppresses
// the spinner (existing --quiet contract). Returns a cleanup func the
// caller must defer unconditionally — a no-op when the pretty branch
// didn't fire, or a Progress.Stop (idempotent) + signal-handler teardown
// when it did.
//
// WR-03 (06-REVIEW.md): indexer.Run/Sync have no ctx parameter to cancel,
// so a SIGINT can't abort the run early — but without a signal handler,
// Go's default SIGINT disposition kills the process before any deferred
// func (including Progress.Stop) ever runs, leaving a stray spinner frame
// on the terminal. Mirror daemon.go's signal.NotifyContext gate and
// explicitly clear the spinner line as soon as the first SIGINT/SIGTERM
// arrives; indexer.Run/Sync itself still runs to completion — a second
// signal falls through to the process's normal (now-restored) signal
// disposition. The returned cleanup func calls stop() (which also cancels
// sigCtx, letting the watcher goroutine exit) before the final,
// now-idempotent prog.Stop().
func startProgress(ctx context.Context, quiet bool, label string) func() {
	if quiet || !present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR")) {
		return func() {}
	}
	prog := present.NewProgress(os.Stderr)
	prog.Start(label)

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCtx.Done()
		prog.Stop() // idempotent — clears the spinner line as soon as the signal arrives
	}()

	return func() {
		stop() // unregisters the handler and cancels sigCtx, ending the goroutine above
		prog.Stop()
	}
}
