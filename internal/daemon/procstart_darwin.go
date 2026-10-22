//go:build darwin

package daemon

import (
	"time"

	"golang.org/x/sys/unix"
)

// processStartTime returns pid's start time as a wall-clock time.Time, read
// from the kernel's process table through sysctl kern.proc.pid (the
// P_starttime field of the pid's kinfo_proc) without CGo. ok is false when
// unix.SysctlKinfoProc returns an error, as it does (EIO) for a pid no process
// holds, or when the start time is zero; isStale then decides on liveness
// alone.
func processStartTime(pid int) (time.Time, bool) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return time.Time{}, false
	}
	start := kp.Proc.P_starttime
	if start.Sec == 0 && start.Usec == 0 {
		return time.Time{}, false
	}
	return time.Unix(start.Unix()).UTC(), true
}
