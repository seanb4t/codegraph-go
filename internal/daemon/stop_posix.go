//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// sendStop delivers a graceful SIGTERM to pid — the same os.FindProcess +
// Signal call shape lock.go's isProcessLive uses, with a real signal
// instead of Signal(0) (D-02, CONTEXT.md "graceful SIGTERM only"
// discretion note). sendStop only delivers the signal; the target process
// is responsible for handling SIGTERM and exiting on its own — this
// function does not wait for or confirm termination.
func sendStop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}
