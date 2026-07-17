package githooks

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs `git <args...>` in dir with deterministic, hermetic flags
// prepended (mirrors internal/gitmeta/fixtures_test.go's runGit — a local
// copy since githooks is a different package, per Task 2 read_first).
// Any failure — including git being absent — skips the calling test.
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

// initRepo creates dir, runs `git init`, and commits a one-line README so
// the repo has a real HEAD.
func initRepo(t *testing.T, dir string) string {
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

// TestMarkerBlock asserts markerBlock() produces the verbatim TS
// sync/git-hooks.js block bytes (D-03): the begin marker, the exact
// subshell-backgrounding guard line (Pitfall 5 — never simplify this), and
// the end marker, joined with "\n".
func TestMarkerBlock(t *testing.T) {
	want := strings.Join([]string{
		"# >>> codegraph sync hook >>>",
		"# Keeps the CodeGraph index fresh while the live file watcher is off",
		"# (e.g. WSL2 /mnt drives). Runs in the background so it never blocks git.",
		"# Managed by codegraph; remove with `codegraph uninit` or delete this block.",
		"if command -v codegraph >/dev/null 2>&1; then",
		"  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1",
		"fi",
		"# <<< codegraph sync hook <<<",
	}, "\n")

	got := markerBlock()
	if got != want {
		t.Fatalf("markerBlock() = %q, want %q", got, want)
	}
	if !strings.Contains(got, markerBegin) {
		t.Errorf("markerBlock() missing begin marker %q", markerBegin)
	}
	if !strings.Contains(got, "if command -v codegraph >/dev/null 2>&1; then") {
		t.Errorf("markerBlock() missing command -v guard line")
	}
	if !strings.Contains(got, "( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1") {
		t.Errorf("markerBlock() missing exact subshell-backgrounding snippet")
	}
	if !strings.Contains(got, markerEnd) {
		t.Errorf("markerBlock() missing end marker %q", markerEnd)
	}
}

func TestStripMarkerBlock_IndentedMarkerStripped(t *testing.T) {
	content := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"  " + markerBegin, // indented — trimmed-line matching must still strip it
		"some block content",
		"  " + markerEnd,
		"echo after",
	}, "\n")

	got := stripMarkerBlock(content)
	want := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"echo after",
	}, "\n")
	if got != want {
		t.Fatalf("stripMarkerBlock(indented) = %q, want %q", got, want)
	}
}

func TestStripMarkerBlock_NoMarkerPassthrough(t *testing.T) {
	content := "#!/bin/sh\necho hi\n"
	got := stripMarkerBlock(content)
	if got != content {
		t.Fatalf("stripMarkerBlock(no marker) = %q, want unchanged %q", got, content)
	}
}

func TestStripMarkerBlock_PreservesSurroundingContent(t *testing.T) {
	content := "#!/bin/sh\n\necho before\n\n" + markerBlock() + "\n\necho after\n"
	got := stripMarkerBlock(content)
	if strings.Contains(got, markerBegin) || strings.Contains(got, markerEnd) {
		t.Fatalf("stripMarkerBlock left markers in output: %q", got)
	}
	if !strings.Contains(got, "echo before") || !strings.Contains(got, "echo after") {
		t.Fatalf("stripMarkerBlock dropped surrounding content: %q", got)
	}
}

func TestIsEffectivelyEmpty_BlankAndShebangOnly(t *testing.T) {
	cases := []string{
		"",
		"\n\n",
		"#!/bin/sh\n",
		"#!/bin/sh\n\n\n",
		"  #!/bin/sh  \n",
	}
	for _, c := range cases {
		if !isEffectivelyEmpty(c) {
			t.Errorf("isEffectivelyEmpty(%q) = false, want true", c)
		}
	}
}

func TestIsEffectivelyEmpty_RealContentIsNotEmpty(t *testing.T) {
	cases := []string{
		"#!/bin/sh\necho hi\n",
		"echo hi\n",
		"#!/bin/sh\n  echo hi  \n",
	}
	for _, c := range cases {
		if isEffectivelyEmpty(c) {
			t.Errorf("isEffectivelyEmpty(%q) = true, want false", c)
		}
	}
}

func TestInstall_FreshRepo_WritesAllThreeHooksWithMode0755(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))

	result := Install(context.Background(), root)

	wantOrder := []string{"post-commit", "post-merge", "post-checkout"}
	if strings.Join(result.Installed, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("InstallResult.Installed = %v, want %v", result.Installed, wantOrder)
	}
	wantContent := "#!/bin/sh\n" + markerBlock() + "\n"
	for _, hook := range wantOrder {
		file := filepath.Join(root, ".git", "hooks", hook)
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", file, err)
		}
		if string(content) != wantContent {
			t.Errorf("%s content = %q, want %q", hook, string(content), wantContent)
		}
		info, err := os.Stat(file)
		if err != nil {
			t.Fatalf("Stat(%s): %v", file, err)
		}
		if info.Mode().Perm() != 0o755 {
			t.Errorf("%s mode = %o, want 0755", hook, info.Mode().Perm())
		}
	}
}

func TestInstall_OverExistingUserHook_PreservesAndAppendsAfterBlankLine(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	userContent := "#!/bin/sh\necho hi\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(userContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	Install(context.Background(), root)

	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := "#!/bin/sh\necho hi\n\n" + markerBlock() + "\n"
	if string(got) != want {
		t.Fatalf("post-commit content = %q, want %q", string(got), want)
	}
}

func TestInstall_ReinstallOnUnmodifiedFile_ByteIdentical(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))

	Install(context.Background(), root)
	first, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile first install: %v", err)
	}

	Install(context.Background(), root)
	second, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile second install: %v", err)
	}

	if string(first) != string(second) {
		t.Fatalf("re-install not byte-identical:\nfirst:  %q\nsecond: %q", first, second)
	}
}

// TestInstall_PriorBlockReplaced_StripThenAppendAtEnd is a deliberate TS
// parity test, not a bug to "simplify" to in-place replacement (Pitfall 2,
// D-02/D-12). Installing over a hook that has content BEFORE and AFTER a
// prior codegraph block strips the old block wherever it sits and
// re-appends the current block at end-of-file — the surviving content
// (before + after, concatenated) gets exactly one fresh block appended,
// never two.
func TestInstall_PriorBlockReplaced_StripThenAppendAtEnd(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	prior := "#!/bin/sh\necho before\n\n" + markerBlock() + "\n\necho after\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(prior), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	Install(context.Background(), root)

	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(got)
	if strings.Count(content, markerBegin) != 1 {
		t.Fatalf("post-commit has %d begin markers, want exactly 1: %q", strings.Count(content, markerBegin), content)
	}
	if !strings.Contains(content, "echo before") || !strings.Contains(content, "echo after") {
		t.Fatalf("post-commit lost surrounding user content: %q", content)
	}
	// The block must now be AFTER "echo after" (moved to end-of-file), not
	// still sitting between "before" and "after" — proving strip-then-append
	// rather than in-place replacement.
	if strings.Index(content, "echo after") > strings.Index(content, markerBegin) {
		t.Fatalf("expected block re-appended at end (after 'echo after'), got: %q", content)
	}
}

func TestInstall_NonRepo_ReturnsSkippedAndWritesNothing(t *testing.T) {
	root := t.TempDir() // not a git repo

	result := Install(context.Background(), root)

	if result.Skipped != "not a git repository" {
		t.Fatalf("Skipped = %q, want %q", result.Skipped, "not a git repository")
	}
	if len(result.Installed) != 0 {
		t.Fatalf("Installed = %v, want empty", result.Installed)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		t.Fatalf(".git directory should not have been created in a non-repo")
	}
}
