package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/agents"
)

// TestMain runs the whole package against a throwaway home: every
// homeEnv variable points under one os.MkdirTemp directory, so a test
// that forgets fakeHome writes there instead of the developer's real
// home. The run stops before any test if os.UserHomeDir() does not
// resolve to that directory. It also sets the ephemeral-binary override,
// because every install this package drives runs from its .test binary;
// a test of the refusal clears it with t.Setenv. It unsets
// forcedColorEnv, so the tests that expect plain output do not depend on
// the developer's shell.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "codegraph-cli-home-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cli: TestMain: MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	if err := isolateHome(home); err != nil {
		fmt.Fprintf(os.Stderr, "cli: TestMain: %v\n", err)
		_ = os.RemoveAll(home)
		os.Exit(1)
	}
	if err := os.Setenv(agents.AllowEphemeralExecPathEnv, "1"); err != nil {
		fmt.Fprintf(os.Stderr, "cli: TestMain: set %s: %v\n", agents.AllowEphemeralExecPathEnv, err)
		_ = os.RemoveAll(home)
		os.Exit(1)
	}
	for _, key := range forcedColorEnv {
		if err := os.Unsetenv(key); err != nil {
			fmt.Fprintf(os.Stderr, "cli: TestMain: unset %s: %v\n", key, err)
			_ = os.RemoveAll(home)
			os.Exit(1)
		}
	}

	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}

// forcedColorEnv lists the variables that style output on a writer that
// is not a terminal. A test that needs one sets it with t.Setenv.
var forcedColorEnv = []string{"CLICOLOR_FORCE", "TTY_FORCE"}

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
