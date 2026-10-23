package daemon

import (
	"fmt"
	"reflect"
	"testing"
)

// captureLog points the logf seam at a recorder and empties logOnce's keys
// for the duration of the test, restoring both in Cleanup. It returns the
// lines logged so far, formatted.
func captureLog(t *testing.T) *[]string {
	t.Helper()
	lines := new([]string)
	prev := logf
	logf = func(format string, args ...any) {
		*lines = append(*lines, fmt.Sprintf(format, args...))
	}
	resetLoggedKeys()
	t.Cleanup(func() {
		logf = prev
		resetLoggedKeys()
	})
	return lines
}

// resetLoggedKeys forgets every key logOnce has logged.
func resetLoggedKeys() {
	loggedMu.Lock()
	defer loggedMu.Unlock()
	clear(loggedKeys)
}

// TestLogOncePerKey: the same key twice logs one line, and a second key logs
// its own line.
func TestLogOncePerKey(t *testing.T) {
	lines := captureLog(t)

	logOnce("a", "first %s", "a")
	logOnce("a", "second %s", "a")
	logOnce("b", "first %s", "b")

	if want := []string{"first a", "first b"}; !reflect.DeepEqual(*lines, want) {
		t.Fatalf("logged %q, want %q", *lines, want)
	}
}
