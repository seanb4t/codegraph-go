//go:build !windows

package daemon

// parentChanged reports whether the current parent pid differs from
// original — the captured-baseline reparent predicate (RESEARCH Pattern
// 5). On Linux (and POSIX subreaper systems generally), when a process's
// immediate parent dies its children are reparented to the nearest living
// subreaper ancestor (e.g. tini, docker --init, or any supervisor using
// PR_SET_CHILD_SUBREAPER) — NOT unconditionally to pid 1. Comparing
// against the captured original ppid, rather than a bare `ppid == 1`
// check, is the robust form that also catches reparenting inside such a
// supervisor/container.
func parentChanged(original int) bool {
	return getppid() != original
}
