//go:build darwin

package daemon

import (
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// withKinfoProc points the kinfoProc seam at a stub that reports start as
// the P_starttime of every pid, and restores the previous value in Cleanup.
func withKinfoProc(t *testing.T, start unix.Timeval) {
	t.Helper()
	prev := kinfoProc
	kinfoProc = func(string, ...int) (*unix.KinfoProc, error) {
		return &unix.KinfoProc{Proc: unix.ExternProc{P_starttime: start}}, nil
	}
	t.Cleanup(func() { kinfoProc = prev })
}

// TestProcessStartTimeDarwinZeroStart: a zero P_starttime for a live pid is
// reported as unavailable, so selfStartedAt never stamps the Unix epoch into
// a record, while a non-zero P_starttime from the same stub is returned as is.
func TestProcessStartTimeDarwinZeroStart(t *testing.T) {
	t.Run("non-zero start time is returned", func(t *testing.T) {
		want := time.Date(2026, 1, 2, 3, 4, 5, 6000, time.UTC)
		withKinfoProc(t, unix.NsecToTimeval(want.UnixNano()))

		got, ok := processStartTime(os.Getpid())
		if !ok || !got.Equal(want) {
			t.Fatalf("processStartTime = %v, ok=%v; want %v, ok=true", got, ok, want)
		}
	})

	t.Run("zero start time is unavailable", func(t *testing.T) {
		withKinfoProc(t, unix.Timeval{})

		if got, ok := processStartTime(os.Getpid()); ok {
			t.Fatalf("processStartTime = %v, ok=true; want ok=false for a zero P_starttime", got)
		}
	})
}
