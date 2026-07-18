package present

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain gates the whole internal/cli/present test package on goleak:
// after every test in this package has run, VerifyTestMain asserts zero
// goroutines remain that were not present at process start — the same
// package-wide convention internal/watch/main_test.go and
// internal/daemon/soak_test.go already use (WR-02). Package-wide rather
// than a per-test defer goleak.VerifyNone(t) so a still-settling
// background goroutine racing a fast-returning test isn't a false
// positive; Progress.Stop's close(stopCh); <-doneCh join already
// guarantees the ticker goroutine has exited by the time Stop returns, so
// this scan passes deterministically instead of via a timing-sensitive
// poll.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
