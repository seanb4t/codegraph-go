package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs `git <args...>` in dir with deterministic, hermetic flags
// prepended (mirrors internal/gitmeta/fixtures_test.go's runGit — a local
// copy since internal/cli is a different package). Any failure — including
// git being absent — skips the calling test.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or fixture unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}

// initGitRepo creates dir, runs `git init`, and commits a one-line README
// so the repo has a real HEAD.
func initGitRepo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	runGit(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("WriteFile README.md: %v", err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

// markerBeginBytes is the verbatim TS sync/git-hooks.js begin marker
// (D-03) — a local copy since internal/githooks doesn't export it; used
// here only as a detection string for on-disk assertions.
const markerBeginBytes = "# >>> codegraph sync hook >>>"

// TestGithooksInstall_RealCobraTree drives `githooks install <repo>`
// through the real command tree (D-13) against a real-git fixture,
// asserting stdout names the installed hooks and the on-disk post-commit
// file contains the marker with mode 0755 (reverting root.go's
// registration or the subcommand wiring turns this red).
func TestGithooksInstall_RealCobraTree(t *testing.T) {
	dir := initGitRepo(t, t.TempDir())

	out, _, err := execCmd("githooks", "install", dir)
	if err != nil {
		t.Fatalf("githooks install: unexpected error: %v", err)
	}
	if !strings.Contains(out, "post-commit") || !strings.Contains(out, "post-merge") || !strings.Contains(out, "post-checkout") {
		t.Fatalf("expected stdout to name the installed hooks, got %q", out)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	info, statErr := os.Stat(hookPath)
	if statErr != nil {
		t.Fatalf("stat post-commit hook: %v", statErr)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("post-commit hook mode = %o, want 0755", info.Mode().Perm())
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read post-commit hook: %v", err)
	}
	if !strings.Contains(string(content), markerBeginBytes) {
		t.Fatalf("expected post-commit hook to contain the begin marker, got %q", string(content))
	}
}

// TestGithooksStatus_AfterInstall_ReportsAllThreeInstalled drives
// `githooks install` then `githooks status` through the real command tree,
// asserting all 3 hooks report installed plus a hooks-dir line, exit 0.
func TestGithooksStatus_AfterInstall_ReportsAllThreeInstalled(t *testing.T) {
	dir := initGitRepo(t, t.TempDir())

	if _, _, err := execCmd("githooks", "install", dir); err != nil {
		t.Fatalf("githooks install: unexpected error: %v", err)
	}

	out, _, err := execCmd("githooks", "status", dir)
	if err != nil {
		t.Fatalf("githooks status: unexpected error: %v", err)
	}
	if !strings.Contains(out, "hooks dir:") {
		t.Fatalf("expected stdout to contain a hooks-dir line, got %q", out)
	}
	for _, hook := range []string{"post-commit", "post-merge", "post-checkout"} {
		if !strings.Contains(out, hook+": installed") {
			t.Fatalf("expected %s to report installed, got %q", hook, out)
		}
	}
}

// TestGithooksRemove_AfterInstall_StripsMarkerFromAllHooks drives
// `githooks install` then `githooks remove` through the real command tree,
// asserting removal is reported and the hook files no longer contain the
// begin marker.
func TestGithooksRemove_AfterInstall_StripsMarkerFromAllHooks(t *testing.T) {
	dir := initGitRepo(t, t.TempDir())

	if _, _, err := execCmd("githooks", "install", dir); err != nil {
		t.Fatalf("githooks install: unexpected error: %v", err)
	}

	out, _, err := execCmd("githooks", "remove", dir)
	if err != nil {
		t.Fatalf("githooks remove: unexpected error: %v", err)
	}
	if !strings.Contains(out, "Removed") {
		t.Fatalf("expected stdout to report removal, got %q", out)
	}

	for _, hook := range []string{"post-commit", "post-merge", "post-checkout"} {
		hookPath := filepath.Join(dir, ".git", "hooks", hook)
		if content, readErr := os.ReadFile(hookPath); readErr == nil {
			if strings.Contains(string(content), markerBeginBytes) {
				t.Fatalf("expected %s to no longer contain the begin marker after remove, got %q", hook, string(content))
			}
		}
		// os.IsNotExist is also an acceptable outcome — the effectively-empty
		// gate deletes the file entirely when only our block was present.
	}
}

// TestGithooksInstall_NonGitDirectory_SkipsCleanlyWithExitZero asserts
// `githooks install` against a non-git directory prints the friendly
// "not a git repository" skip message and returns a nil error (exit 0),
// never a hard error.
func TestGithooksInstall_NonGitDirectory_SkipsCleanlyWithExitZero(t *testing.T) {
	dir := t.TempDir()

	out, _, err := execCmd("githooks", "install", dir)
	if err != nil {
		t.Fatalf("githooks install (non-repo): expected nil error, got %v", err)
	}
	if !strings.Contains(out, "not a git repository") {
		t.Fatalf("expected stdout to contain the not-a-git-repository skip message, got %q", out)
	}
}
