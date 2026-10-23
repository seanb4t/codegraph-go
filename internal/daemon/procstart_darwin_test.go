//go:build darwin

package daemon

import (
	"os"
	"strings"
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

// withKinfoProcError points the kinfoProc seam at a stub that fails every
// read with err, restores the previous value in Cleanup, and returns the
// number of reads made through it.
func withKinfoProcError(t *testing.T, err error) *int {
	t.Helper()
	calls := new(int)
	prev := kinfoProc
	kinfoProc = func(string, ...int) (*unix.KinfoProc, error) {
		*calls++
		return nil, err
	}
	t.Cleanup(func() { kinfoProc = prev })
	return calls
}

// TestRegistryListLogsDeniedStartTimeReadOnce: when the kernel denies every
// start-time read, liveness decides, so a valid record for a live pid is
// listed by each of two listings, and the process logs the error once. EIO,
// which means no process holds the pid, logs nothing.
func TestRegistryListLogsDeniedStartTimeReadOnce(t *testing.T) {
	rec := Record{PID: os.Getpid(), StartedAt: time.Now().UTC().Truncate(time.Second), RepoRoot: "/live/repo"}
	listTwice := func(t *testing.T) {
		t.Helper()
		for range 2 {
			got, _, err := List()
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(got) != 1 || got[0] != rec {
				t.Fatalf("List: got %+v, want exactly %+v", got, rec)
			}
		}
	}

	t.Run("a denied read is logged once", func(t *testing.T) {
		withRegistryDir(t)
		lines := captureLog(t)
		calls := withKinfoProcError(t, unix.EPERM)
		if err := Register(rec); err != nil {
			t.Fatalf("Register: %v", err)
		}

		listTwice(t)

		if *calls != 2 {
			t.Fatalf("kinfoProc ran %d times, want once per listing (2)", *calls)
		}
		if len(*lines) != 1 || !strings.Contains((*lines)[0], unix.EPERM.Error()) {
			t.Fatalf("logged %q, want one line naming %q", *lines, unix.EPERM.Error())
		}
	})

	t.Run("EIO is not logged", func(t *testing.T) {
		withRegistryDir(t)
		lines := captureLog(t)
		calls := withKinfoProcError(t, unix.EIO)
		if err := Register(rec); err != nil {
			t.Fatalf("Register: %v", err)
		}

		listTwice(t)

		if *calls != 2 {
			t.Fatalf("kinfoProc ran %d times, want once per listing (2)", *calls)
		}
		if len(*lines) != 0 {
			t.Fatalf("logged %q, want nothing", *lines)
		}
	})
}
