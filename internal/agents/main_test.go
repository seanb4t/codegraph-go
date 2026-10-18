package agents

import (
	"fmt"
	"os"
	"testing"
)

// TestMain runs the whole package against a throwaway home: every
// homeEnv variable points under one os.MkdirTemp directory, so a test
// that forgets fakeHome writes there instead of the developer's real
// home. The run stops before any test if os.UserHomeDir() does not
// resolve to that directory.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "codegraph-agents-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "agents: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	if err := isolateHome(home); err != nil {
		fmt.Fprintf(os.Stderr, "agents: TestMain: %v\n", err)
		_ = os.RemoveAll(home)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
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
