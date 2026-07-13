// Package bench holds the pure-logic measurement core for the codegraph
// benchmark harness (PERF-01, INDX-06). Peak RSS is always measured
// externally, via the OS-level rusage of a child process the caller
// spawned (exec.Cmd.ProcessState.SysUsage()) — never via in-process Go
// runtime memory statistics, which cannot be compared fairly against the
// TS Node process (D-05). This package has no network or crypto surface
// and does not shell out itself; callers own the exec.Cmd. Measurement
// helpers never panic — they return (T, error) so a single bad run fails
// its CI step loudly instead of crashing the whole gate.
package bench

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
)

// PeakRSSBytes returns the peak resident set size, in bytes, of the
// process described by state. state must come from a completed
// exec.Cmd (i.e. after Wait() returns) so ProcessState.SysUsage() is
// populated. The raw ru_maxrss value is normalized to bytes based on
// runtime.GOOS (see normalizeMaxrss) since its unit differs by OS.
func PeakRSSBytes(state *os.ProcessState) (int64, error) {
	ru, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return 0, fmt.Errorf("bench: platform does not expose syscall.Rusage")
	}
	return normalizeMaxrss(runtime.GOOS, int64(ru.Maxrss))
}

// normalizeMaxrss converts a raw ru_maxrss value to bytes based on goos.
// Linux reports ru_maxrss in kilobytes; Darwin (BSD-derived) reports it
// already in bytes. Any other OS is unsupported for RSS measurement and
// returns a loud error rather than a silently wrong or zero value.
func normalizeMaxrss(goos string, rawMaxrss int64) (int64, error) {
	switch goos {
	case "linux":
		return rawMaxrss * 1024, nil
	case "darwin":
		return rawMaxrss, nil
	default:
		return 0, fmt.Errorf("bench: unsupported OS for RSS measurement: %s", goos)
	}
}
