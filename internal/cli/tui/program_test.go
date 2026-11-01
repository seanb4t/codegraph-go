package tui

import (
	"bytes"
	"io"
	"regexp"
	"testing"
	"time"

	"github.com/charmbracelet/x/exp/teatest/v2"
)

const (
	altScreenEnter = "\x1b[?1049h"
	altScreenLeave = "\x1b[?1049l"

	// programDeadline bounds how long a test waits for a frame or for the
	// Program to exit. It only fires on failure.
	programDeadline = 10 * time.Second
)

var (
	// csiSeq matches one CSI control sequence (ECMA-48: parameter bytes,
	// intermediate bytes, one final byte).
	csiSeq = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

	// modeQuery matches a DECRQM request (CSI ? Ps $ p). A reply to one that
	// arrives after the Program has exited is echoed into the user's shell
	// (G-07-1).
	modeQuery = regexp.MustCompile(`\x1b\[\??[0-9;]*\$p`)

	// privateModesOnly matches a run of DEC private mode set/reset sequences
	// (CSI ? Ps h / CSI ? Ps l) and nothing else.
	privateModesOnly = regexp.MustCompile(`^(?:\x1b\[\?[0-9;]*[hl])*$`)
)

// textOf returns b with every CSI sequence removed: the bytes a terminal
// would print as text.
func textOf(b []byte) string {
	return string(csiSeq.ReplaceAll(b, nil))
}

// waitForFrame reads tm's output until it contains want and returns every
// byte read so far. It fails the test if want does not appear before
// programDeadline.
func waitForFrame(t *testing.T, tm *teatest.TestModel, want string) []byte {
	t.Helper()
	var seen []byte
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		seen = append(seen[:0], b...)
		return bytes.Contains(b, []byte(want))
	}, teatest.WithDuration(programDeadline), teatest.WithCheckInterval(time.Millisecond))
	return seen
}

// finalOutput waits for tm's Program to exit and returns the output bytes
// not yet read.
func finalOutput(t *testing.T, tm *teatest.TestModel) []byte {
	t.Helper()
	out, err := io.ReadAll(tm.FinalOutput(t, teatest.WithFinalTimeout(programDeadline)))
	if err != nil {
		t.Fatalf("read program output: %v", err)
	}
	return out
}
