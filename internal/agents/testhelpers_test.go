package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
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
	for key, value := range homeEnv(home) {
		t.Setenv(key, value)
	}
	return home
}

// homeEnv returns every home-derived env var this package's targets
// consult, pointed under home. fakeHome and TestMain both set exactly
// this set; a new agent target that reads another variable adds it here.
func homeEnv(home string) map[string]string {
	return map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"XDG_STATE_HOME":  filepath.Join(home, ".local", "state"),
		"HERMES_HOME":     filepath.Join(home, ".hermes"),
	}
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

// assertSymlinkTo fails unless link is still a symlink with that target.
func assertSymlinkTo(t *testing.T, link, target string) {
	t.Helper()
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("%s is no longer a symlink: %v", link, err)
	}
	if got != target {
		t.Fatalf("%s links to %q, want %q", link, got, target)
	}
}

// projectAndOutside makes a project directory and a directory outside it,
// and makes the project the working directory.
func projectAndOutside(t *testing.T) (project, outside string) {
	t.Helper()
	fakeHome(t)
	base := realpathtest.TempDir(t)
	project, outside = filepath.Join(base, "project"), filepath.Join(base, "outside")
	for _, dir := range []string{project, outside} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(project)
	return project, outside
}

func errorNaming(result WriteResult, parts ...string) bool {
	for _, err := range result.Errors {
		all := true
		for _, part := range parts {
			all = all && strings.Contains(err.Error(), part)
		}
		if all {
			return true
		}
	}
	return false
}

func noteNaming(result WriteResult, part string) bool {
	for _, note := range result.Notes {
		if strings.Contains(note, part) {
			return true
		}
	}
	return false
}

// realHome is fakeHome with the home's own symlinks resolved.
func realHome(t *testing.T) string {
	t.Helper()
	home, err := filepath.EvalSymlinks(fakeHome(t))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range homeEnv(home) {
		t.Setenv(key, value)
	}
	return home
}

func assertNoErrors(t *testing.T, result WriteResult) {
	t.Helper()
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s exists (or cannot be read: %v), want no file", path, err)
	}
}
