//go:build !linux && !darwin

package daemon

import "time"

// processStartTime is unavailable on platforms other than Linux and macOS
// (WR-02). isStale falls back to liveness-only staleness detection there:
// ok is always false, never a false claim of corroboration.
func processStartTime(pid int) (time.Time, bool) {
	return time.Time{}, false
}
