//go:build windows

package daemon

import "os"

// sendStop hard-kills pid on Windows (os.Process.Kill, which the runtime
// implements via TerminateProcess). Windows has no cross-process
// SIGTERM-equivalent: os.Process.Signal on Windows only accepts os.Kill,
// any other signal value returns an error (RESEARCH Open Question #1,
// Common Pitfall 3) — so unlike stop_posix.go's graceful SIGTERM, this is
// an explicit, documented divergence, accepted for v1.0 (Assumption A4)
// rather than a silent no-op or panic. The registry/lockfile self-heal
// (isStale in lock.go) on the next scan regardless of how ungracefully the
// process exited, so no on-disk state is left corrupted by the hard-kill.
// Deliberately uses only stdlib os.Process.Kill — no golang.org/x/sys
// import — mirroring locked_windows.go's minimal-direct-wrapper style.
func sendStop(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
