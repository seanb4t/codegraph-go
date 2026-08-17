package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeHome points HOME (and every per-agent home-derived env var this
// package's targets consult — XDG_CONFIG_HOME for opencode, HERMES_HOME
// for Hermes) at fresh subdirectories of one isolated t.TempDir(), so
// every agent's global-scope test runs against a fake home rather than
// the real developer machine, mocking os.homedir() per test. Returns the
// fake home root.
func fakeHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("HERMES_HOME", filepath.Join(home, ".hermes"))
	return home
}

// readFile reads path and fails the test on error, for terse test-body
// assertions against a written config/instructions file.
func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile(%s): %v", path, err)
	}
	return string(data)
}

// writeFile writes content to path, creating parent directories as
// needed, and fails the test on error — used to seed pre-existing
// fixture state before exercising a surgical write helper.
func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("writeFile mkdir(%s): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile(%s): %v", path, err)
	}
}
