//go:build darwin

package daemon

import (
	"errors"
	"time"

	"golang.org/x/sys/unix"
)

// kinfoProc is unix.SysctlKinfoProc indirected behind a package-level func
// var, the same test-seam convention as startTimeOf, so tests can supply a
// kinfo_proc the kernel would not return for a live test process.
var kinfoProc = unix.SysctlKinfoProc

// startTimeReadKey is the logOnce key of a failed start-time read. It is one
// key for every pid, because a denied sysctl denies every pid alike.
const startTimeReadKey = "kern.proc.pid"

// processStartTime returns pid's start time as a wall-clock time.Time, read
// from the kernel's process table through sysctl kern.proc.pid (the
// P_starttime field of the pid's kinfo_proc) without CGo. ok is false when
// unix.SysctlKinfoProc returns an error, or when the start time is zero;
// isStale then decides on liveness alone. EIO, which SysctlKinfoProc returns
// for a pid no process holds, is silent. Any other error, such as a read a
// sandbox denies, turns pid-reuse detection off for every record, so it is
// logged once per process.
func processStartTime(pid int) (time.Time, bool) {
	kp, err := kinfoProc("kern.proc.pid", pid)
	if err != nil {
		if !errors.Is(err, unix.EIO) {
			logOnce(startTimeReadKey, "daemon: reading process start times: %v; a recycled pid cannot be detected", err)
		}
		return time.Time{}, false
	}
	start := kp.Proc.P_starttime
	if start.Sec == 0 && start.Usec == 0 {
		return time.Time{}, false
	}
	return time.Unix(start.Unix()).UTC(), true
}
