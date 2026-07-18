//go:build windows

package daemon

import "golang.org/x/sys/windows"

// parentChanged reports whether the process identified by original
// (captured once, at daemon start, via os.Getppid()'s cross-platform
// value on Windows) has gone away, via parentAlive. Windows has no
// getppid-reparent semantics (no subreaper concept, and a dead parent's
// pid is simply gone, never reassigned to a live ancestor the way POSIX
// reparents to init/a subreaper) — so unlike watchdog_posix.go's
// parentChanged, this is a liveness check, not a "did the value change"
// check.
func parentChanged(original int) bool {
	return !parentAlive(original)
}

// parentAlive probes whether the process identified by pid is still
// running, via OpenProcess(SYNCHRONIZE)+WaitForSingleObject(0):
// WAIT_TIMEOUT means the process is still running, WAIT_OBJECT_0 means it
// has exited. Source: golang.org/x/sys/windows (promoted to a direct
// require by this file); pattern confirmed against Microsoft's own
// OpenProcess+WaitForSingleObject documentation (07-RESEARCH.md Code
// Examples "Windows parent liveness").
//
// GetExitCodeProcess's STILL_ACTIVE sentinel is deliberately NOT used
// here — Microsoft's own documentation notes STILL_ACTIVE is unreliable
// for liveness checks (a process can legitimately exit with code 259,
// making STILL_ACTIVE ambiguous between "still running" and "exited with
// that exact code"), the same caveat internal/graphstore/locked_windows.go
// documents for its own minimal-syscall Windows divergence.
func parentAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return false // can't open — treat as gone
	}
	defer windows.CloseHandle(h)
	ev, err := windows.WaitForSingleObject(h, 0)
	return err == nil && ev == uint32(windows.WAIT_TIMEOUT)
}
