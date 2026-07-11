//go:build !linux

package daemon

import "time"

// processStartTime is unavailable on non-Linux platforms in v1 (WR-02):
// there is no equivalent of Linux's /proc/<pid>/stat this package reads
// without a CGo dependency (e.g. macOS's sysctl KERN_PROC or Windows'
// GetProcessTimes both require syscalls this package deliberately doesn't
// wrap yet). isStale falls back to liveness-only staleness detection on
// these platforms, exactly as it did before WR-02's corroboration was
// added — ok is always false, never a false claim of corroboration.
func processStartTime(pid int) (time.Time, bool) {
	return time.Time{}, false
}
