package watch

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain gates the whole internal/watch test package on goleak: after
// every test in this package has run, VerifyTestMain asserts zero
// goroutines remain that were not present at process start (SYNC-06). This
// is the package-wide guard TestSoak's many watch->debounce cycles must
// pass — package-wide rather than a per-test defer goleak.VerifyNone(t)
// because a still-scheduled timer callback racing a fast-returning test
// would otherwise be a false-positive leak (RESEARCH Pattern 7).
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
