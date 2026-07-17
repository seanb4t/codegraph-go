package cli

import (
	"strings"
	"testing"
)

// TestInitAdvisory_WatcherEnabled asserts init's D-07 advisory prints
// NOTHING when the watcher runs normally (the default on this machine:
// not WSL2, CODEGRAPH_NO_WATCH unset) — the not-always-on guarantee that
// makes HOOK-03's "narrower trigger" real. Drives the real cobra `init`
// command end-to-end (D-13); reverting printWatchFallbackAdvisory's wiring
// would not turn this one red on its own, but its sibling
// TestInitAdvisory_WatcherDisabled below does.
func TestInitAdvisory_WatcherEnabled(t *testing.T) {
	// Make the "watcher enabled" precondition explicit rather than ambient
	// (IN-02): a stray exported CODEGRAPH_NO_WATCH in a developer's shell
	// would otherwise silently flip what this test asserts without it
	// failing loudly in an attributable way.
	t.Setenv("CODEGRAPH_NO_WATCH", "")

	dir := copyFixture(t)

	out, _, err := execCmd("init", dir)
	if err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	if strings.Contains(out, "Live file watching is disabled") {
		t.Fatalf("expected no watch-fallback advisory when the watcher is enabled, got:\n%s", out)
	}
}

// TestInitAdvisory_WatcherDisabled forces "disabled" via the injectable
// watch.Probe seam (CODEGRAPH_NO_WATCH=1, which a nil Probe.Env resolves
// through os.Getenv) against a real-git fixture with no hooks installed,
// and asserts init's stdout contains both the disabled warning and the
// `codegraph githooks install` pointer (D-07 steps 1-2, 4-5). Reverting
// init.go's printWatchFallbackAdvisory wiring turns this test red (D-13).
func TestInitAdvisory_WatcherDisabled(t *testing.T) {
	t.Setenv("CODEGRAPH_NO_WATCH", "1")

	dir := copyFixture(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	out, _, err := execCmd("init", dir)
	if err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Live file watching is disabled here") {
		t.Fatalf("expected the disabled-watcher warning, got:\n%s", out)
	}
	if !strings.Contains(out, "codegraph githooks install") {
		t.Fatalf("expected the githooks-install pointer, got:\n%s", out)
	}
}

// TestInitAdvisory_WatcherDisabled_NonGitRepo asserts the D-07 step-3
// branch: watcher disabled + not a git repo prints the manual
// `codegraph sync` hint instead of any hook-related pointer.
func TestInitAdvisory_WatcherDisabled_NonGitRepo(t *testing.T) {
	t.Setenv("CODEGRAPH_NO_WATCH", "1")

	dir := copyFixture(t)

	out, _, err := execCmd("init", dir)
	if err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Run `codegraph sync` after changing files to refresh the index.") {
		t.Fatalf("expected the manual sync hint for a non-repo target, got:\n%s", out)
	}
	if strings.Contains(out, "githooks install") {
		t.Fatalf("expected no githooks pointer for a non-repo target, got:\n%s", out)
	}
}

// TestInitAdvisory_WatcherDisabled_HooksAlreadyInstalled asserts the D-07
// step-4 branch: watcher disabled + git repo + hooks already installed
// reports the already-installed info line instead of the install pointer.
func TestInitAdvisory_WatcherDisabled_HooksAlreadyInstalled(t *testing.T) {
	dir := copyFixture(t)
	runGit(t, dir, "init")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")

	if _, _, err := execCmd("githooks", "install", dir); err != nil {
		t.Fatalf("githooks install: unexpected error: %v", err)
	}

	t.Setenv("CODEGRAPH_NO_WATCH", "1")

	out, _, err := execCmd("init", dir)
	if err != nil {
		t.Fatalf("init: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Git sync hooks are already installed") {
		t.Fatalf("expected the already-installed info line, got:\n%s", out)
	}
	if strings.Contains(out, "codegraph githooks install") {
		t.Fatalf("expected no install pointer when hooks are already installed, got:\n%s", out)
	}
}
