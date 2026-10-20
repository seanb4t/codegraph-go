package daemon

import (
	"fmt"
	"os"
	"testing"

	"go.uber.org/goleak"
)

// TestMain runs the whole package against a throwaway home and gates it on
// goleak. Before any test runs, every homeEnv variable points under one
// os.MkdirTemp directory, so a test that drives Run, Register or
// Deregister without withRegistryDir writes its records there instead of
// the developer's real $XDG_STATE_HOME/codegraph/daemons. The run stops
// before any test if os.UserHomeDir() does not resolve to that directory.
//
// After every test has run, VerifyTestMain asserts zero goroutines remain
// that were not present at process start (SYNC-06). Its Cleanup option
// replaces goleak's own os.Exit: it removes the home, then exits with the
// code goleak computed, so a leak still fails the run.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "codegraph-daemon-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	if err := isolateHome(home); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: TestMain: %v\n", err)
		_ = os.RemoveAll(home)
		os.Exit(1)
	}

	goleak.VerifyTestMain(m, goleak.Cleanup(func(code int) {
		_ = os.RemoveAll(home)
		os.Exit(code)
	}))
}

// isolateHome sets every homeEnv variable under home and confirms that
// os.UserHomeDir() now resolves there.
func isolateHome(home string) error {
	for key, value := range homeEnv(home) {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	got, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("os.UserHomeDir: %w", err)
	}
	if got != home {
		return fmt.Errorf("os.UserHomeDir() = %q, want the fake home %q", got, home)
	}
	return nil
}
