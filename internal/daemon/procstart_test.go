//go:build linux || darwin

package daemon

import (
	"runtime"
	"testing"
	"time"
)

// TestProcessStartTime: processStartTime reports a child's start between the
// instants just before and just after it was started, and ok=false for a pid
// no process holds. Linux start times read up to two seconds early (btime and
// the tick count are each truncated), so the window opens two seconds earlier
// there.
func TestProcessStartTime(t *testing.T) {
	early := time.Millisecond
	if runtime.GOOS == "linux" {
		early = 2 * time.Second
	}

	before := time.Now()
	cmd := liveProcess(t)
	after := time.Now()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	start, ok := processStartTime(cmd.Process.Pid)
	if !ok {
		t.Fatalf("processStartTime(child pid %d): ok=false, want its start time", cmd.Process.Pid)
	}
	if lo, hi := before.Add(-early), after.Add(time.Millisecond); start.Before(lo) || start.After(hi) {
		t.Fatalf("processStartTime(child pid %d) = %v, want within [%v, %v]", cmd.Process.Pid, start, lo, hi)
	}

	dead := deadPID(t)
	if got, ok := processStartTime(dead); ok {
		t.Fatalf("processStartTime(dead pid %d) = %v, ok=true; want ok=false", dead, got)
	}
}
