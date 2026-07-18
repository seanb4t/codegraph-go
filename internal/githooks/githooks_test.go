package githooks

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	got, ok := stripMarkerBlock(content)
	want := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"echo after",
	}, "\n")
	if !ok {
		t.Fatalf("stripMarkerBlock(indented) ok = false, want true")
	}
	if got != want {
		t.Fatalf("stripMarkerBlock(indented) = %q, want %q", got, want)
	}
}

func TestStripMarkerBlock_NoMarkerPassthrough(t *testing.T) {
	content := "#!/bin/sh\necho hi\n"
	got, ok := stripMarkerBlock(content)
	if !ok {
		t.Fatalf("stripMarkerBlock(no marker) ok = false, want true")
	}
	if got != content {
		t.Fatalf("stripMarkerBlock(no marker) = %q, want unchanged %q", got, content)
	}
}

func TestStripMarkerBlock_PreservesSurroundingContent(t *testing.T) {
	content := "#!/bin/sh\n\necho before\n\n" + markerBlock() + "\n\necho after\n"
	got, ok := stripMarkerBlock(content)
	if !ok {
		t.Fatalf("stripMarkerBlock ok = false, want true")
	}
	if strings.Contains(got, markerBegin) || strings.Contains(got, markerEnd) {
		t.Fatalf("stripMarkerBlock left markers in output: %q", got)
	}
	if !strings.Contains(got, "echo before") || !strings.Contains(got, "echo after") {
		t.Fatalf("stripMarkerBlock dropped surrounding content: %q", got)
	}
}

// TestStripMarkerBlock_UnterminatedBegin_ReturnsUnchanged is the CR-01
// regression test: a begin marker with no matching end marker must not be
// treated as "block extends to EOF" (TS's inherited data-loss bug) —
// stripMarkerBlock must report ok=false and hand back the content
// untouched so callers know not to trust the strip.
func TestStripMarkerBlock_UnterminatedBegin_ReturnsUnchanged(t *testing.T) {
	content := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		markerBegin,
		"... (end marker missing) ...",
		"echo after",
		"echo more-user-content",
	}, "\n")

	got, ok := stripMarkerBlock(content)
	if ok {
		t.Fatalf("stripMarkerBlock(unterminated begin) ok = true, want false")
	}
	if got != content {
		t.Fatalf("stripMarkerBlock(unterminated begin) = %q, want unchanged %q", got, content)
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

// TestInstall_ReinstallOnUnmodifiedFile_ByteIdentical asserts Install
// converges to a stable fixed point: once a hook file has round-tripped
// through Install at least once, re-installing again produces
// byte-identical output. This is verified from the SECOND install onward,
// not the very first-vs-second transition — verbatim TS installGitSyncHook
// has a documented quirk (see the package-level note on Install) where the
// from-scratch seed form ("#!/bin/sh\n"+block, no blank-line separator)
// differs by exactly one blank line from the round-tripped form produced
// once the file already exists ("#!/bin/sh\n\n"+block, base+separator
// path) — because the surviving shebang line becomes non-empty "base"
// content once stripMarkerBlock sees it. From the second install onward
// the round-tripped form is a genuine fixed point (re-install never
// changes it again), which is what "idempotent" means here.
func TestInstall_ReinstallOnUnmodifiedFile_ByteIdentical(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))

	Install(context.Background(), root) // seeds the from-scratch form
	Install(context.Background(), root) // first round-trip: converges to the stable form
	second, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile second install: %v", err)
	}

	Install(context.Background(), root) // re-install on the now-stable form
	third, err := os.ReadFile(filepath.Join(root, ".git", "hooks", "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile third install: %v", err)
	}

	if string(second) != string(third) {
		t.Fatalf("re-install not byte-identical at steady state:\nsecond: %q\nthird:  %q", second, third)
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

// TestInstall_HooksPathIsFile_ReturnsSkippedCouldNotAccess exercises the
// "could not access the git hooks directory" skip branch (WR-03): pre-seed
// a regular FILE at the exact path git reports as the hooks directory, so
// os.MkdirAll(hooksDir, ...) fails because a non-directory already
// occupies that path.
func TestInstall_HooksPathIsFile_ReturnsSkippedCouldNotAccess(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.RemoveAll(hooksDir); err != nil {
		t.Fatalf("RemoveAll(%s): %v", hooksDir, err)
	}
	if err := os.WriteFile(hooksDir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", hooksDir, err)
	}

	result := Install(context.Background(), root)

	if result.Skipped != "could not access the git hooks directory" {
		t.Fatalf("Skipped = %q, want %q", result.Skipped, "could not access the git hooks directory")
	}
	if len(result.Installed) != 0 {
		t.Fatalf("Installed = %v, want empty", result.Installed)
	}
}

// TestInstall_OneHookPathIsDirectory_PartialSuccessWithErrors exercises
// the WR-01/WR-03 partial-failure path: pre-seed a directory (not a file)
// at post-commit's target path so fsatomic.WriteFile fails to rename onto
// it (can't rename a regular file over a directory), while post-merge and
// post-checkout — ordinary absent files — succeed normally. Asserts
// Installed contains only the two that succeeded and Errors has exactly
// one entry naming the failed hook.
func TestInstall_OneHookPathIsDirectory_PartialSuccessWithErrors(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(filepath.Join(hooksDir, "post-commit"), 0o755); err != nil {
		t.Fatalf("MkdirAll(post-commit as dir): %v", err)
	}

	result := Install(context.Background(), root)

	wantInstalled := []string{"post-merge", "post-checkout"}
	if strings.Join(result.Installed, ",") != strings.Join(wantInstalled, ",") {
		t.Fatalf("Installed = %v, want %v", result.Installed, wantInstalled)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("Errors = %v, want exactly 1 entry", result.Errors)
	}
	if !strings.Contains(result.Errors[0].Error(), "post-commit") {
		t.Errorf("Errors[0] = %v, want it to name post-commit", result.Errors[0])
	}
}

// TestInstall_EditThenRemove_ByteInvariant is the D-16/TEST-03 regression
// test: the genuine gap TEST-03's RESEARCH identified — the pre-existing
// TestRemove_WithUserContent_PreservesRemainderBytes only covers *remove*
// preserving a hand-written remainder, never the full install -> edit ->
// remove round trip. This proves that install -> a user hand-edit OUTSIDE
// the marker block (a new line added to the surviving base content, never
// touching the codegraph-managed block itself) -> remove returns the hook
// file byte-identical to the pre-install original PLUS the user's edit,
// with the marker block fully stripped. Also backstops install->install
// (fixed point) and remove->remove (clean no-op) against this same
// fixture, per TEST-03's idempotency edge.
func TestInstall_EditThenRemove_ByteInvariant(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	file := filepath.Join(hooksDir, "post-commit")

	original := "#!/bin/sh\necho original-user-content\n"
	if err := os.WriteFile(file, []byte(original), 0o755); err != nil {
		t.Fatalf("WriteFile pristine original: %v", err)
	}

	Install(context.Background(), root)

	installed, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after install: %v", err)
	}
	wantInstalled := "#!/bin/sh\necho original-user-content\n\n" + markerBlock() + "\n"
	if string(installed) != wantInstalled {
		t.Fatalf("post-commit after install = %q, want %q", string(installed), wantInstalled)
	}

	// Simulate a user hand-editing the installed hook: a new line added to
	// their own content, OUTSIDE the marker block — the block itself is
	// never touched.
	userEdit := "#!/bin/sh\necho original-user-content\necho user-added-after-install\n\n" + markerBlock() + "\n"
	if err := os.WriteFile(file, []byte(userEdit), 0o755); err != nil {
		t.Fatalf("WriteFile user edit: %v", err)
	}

	Remove(context.Background(), root)

	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after remove: %v", err)
	}
	want := "#!/bin/sh\necho original-user-content\necho user-added-after-install\n"
	if string(got) != want {
		t.Fatalf("post-commit after install->edit->remove = %q, want byte-identical to pre-install original + user edit %q", string(got), want)
	}

	// Idempotency backstop: install->install on this now-user-content-only
	// file is a fixed point (no stale block to reintroduce blank-line
	// drift), and remove->remove is a clean no-op.
	Install(context.Background(), root)
	afterReinstall, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after re-install: %v", err)
	}
	Install(context.Background(), root)
	afterReinstallAgain, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after second re-install: %v", err)
	}
	if string(afterReinstall) != string(afterReinstallAgain) {
		t.Fatalf("install->install not a fixed point:\nfirst:  %q\nsecond: %q", afterReinstall, afterReinstallAgain)
	}

	firstRemove := Remove(context.Background(), root)
	removedAfterReinstall := false
	for _, h := range firstRemove.Removed {
		if h == "post-commit" {
			removedAfterReinstall = true
		}
	}
	if !removedAfterReinstall {
		t.Fatalf("Remove after re-install: Removed = %v, want post-commit included", firstRemove.Removed)
	}
	secondRemove := Remove(context.Background(), root)
	if len(secondRemove.Removed) != 0 {
		t.Fatalf("second Remove() should be a no-op: Removed = %v", secondRemove.Removed)
	}
}

func TestRemove_WithUserContent_PreservesRemainderBytes(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	withUser := "#!/bin/sh\necho hi\n\n" + markerBlock() + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(withUser), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	found := false
	for _, h := range result.Removed {
		if h == "post-commit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Removed = %v, want post-commit included", result.Removed)
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile after remove: %v", err)
	}
	want := "#!/bin/sh\necho hi\n"
	if string(got) != want {
		t.Fatalf("post-commit after remove = %q, want %q", string(got), want)
	}
}

func TestRemove_EffectivelyEmptyRemainder_DeletesFile(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	Install(context.Background(), root)
	file := filepath.Join(root, ".git", "hooks", "post-commit")

	result := Remove(context.Background(), root)

	found := false
	for _, h := range result.Removed {
		if h == "post-commit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Removed = %v, want post-commit included", result.Removed)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("post-commit should have been deleted (effectively empty), stat err = %v", err)
	}
}

func TestRemove_NeverInstalled_UntouchedNoError(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	userOnly := "#!/bin/sh\necho untouched\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(userOnly), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Removed (never installed): %v", result.Removed)
		}
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != userOnly {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), userOnly)
	}
}

func TestRemove_Twice_SecondRunIsNoOp(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	Install(context.Background(), root)

	first := Remove(context.Background(), root)
	if len(first.Removed) == 0 {
		t.Fatalf("first Remove() = %v, want non-empty Removed", first.Removed)
	}

	second := Remove(context.Background(), root)
	if len(second.Removed) != 0 {
		t.Fatalf("second Remove() = %v, want empty (already removed)", second.Removed)
	}
}

// TestRemove_UnwritableHooksDir_AccumulatesErrors exercises the WR-01/
// WR-03 partial-failure path on Remove: after installing all three hooks
// normally, make the hooks directory unwritable so the strip-and-rewrite
// path's fsatomic.WriteFile/os.Remove calls fail, and assert the failures
// are accumulated into Errors rather than silently discarded. Skipped
// under root (permission bits don't apply) and on Windows (no POSIX
// permission model), matching the existing skip convention in
// internal/upgrade's permission-based tests.
func TestRemove_UnwritableHooksDir_AccumulatesErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	Install(context.Background(), root)

	if err := os.Chmod(hooksDir, 0o500); err != nil {
		t.Fatalf("Chmod(%s, 0500): %v", hooksDir, err)
	}
	t.Cleanup(func() { _ = os.Chmod(hooksDir, 0o755) })

	result := Remove(context.Background(), root)

	if len(result.Removed) != 0 {
		t.Fatalf("Removed = %v, want empty (writes should have failed)", result.Removed)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("Errors = %v, want at least one accumulated error", result.Errors)
	}
}

// TestRemove_TSInstalledBlock_DetectedAndRemovable pastes the verbatim TS
// sync/git-hooks.js marker block bytes into a hook file (as if TS
// CodeGraph, not this Go binary, had installed it) and asserts Status
// detects it as installed and Remove successfully strips it — the D-12
// cross-tool compatibility fixture (D-03).
func TestRemove_TSInstalledBlock_DetectedAndRemovable(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	tsBlock := strings.Join([]string{
		"# >>> codegraph sync hook >>>",
		"# Keeps the CodeGraph index fresh while the live file watcher is off",
		"# (e.g. WSL2 /mnt drives). Runs in the background so it never blocks git.",
		"# Managed by codegraph; remove with `codegraph uninit` or delete this block.",
		"if command -v codegraph >/dev/null 2>&1; then",
		"  ( codegraph sync >/dev/null 2>&1 & ) >/dev/null 2>&1",
		"fi",
		"# <<< codegraph sync hook <<<",
	}, "\n")
	tsInstalled := "#!/bin/sh\n" + tsBlock + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(tsInstalled), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	status := Status(context.Background(), root)
	found := false
	for _, h := range status.Hooks {
		if h.Name == "post-commit" {
			if !h.Installed {
				t.Fatalf("Status: post-commit Installed = false, want true (TS-installed block must be detected)")
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("Status.Hooks missing post-commit entry: %v", status.Hooks)
	}

	result := Remove(context.Background(), root)
	removed := false
	for _, h := range result.Removed {
		if h == "post-commit" {
			removed = true
		}
	}
	if !removed {
		t.Fatalf("Remove did not remove the TS-installed post-commit block: %v", result.Removed)
	}
}

func TestStatus_MixedInstalledState_ReportsPerHook(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	installedContent := "#!/bin/sh\n" + markerBlock() + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(installedContent), 0o755); err != nil {
		t.Fatalf("WriteFile post-commit: %v", err)
	}
	// post-merge and post-checkout are left absent (not installed).

	status := Status(context.Background(), root)

	if len(status.Hooks) != 3 {
		t.Fatalf("Status.Hooks len = %d, want 3", len(status.Hooks))
	}
	wantOrder := []string{"post-commit", "post-merge", "post-checkout"}
	wantInstalled := map[string]bool{"post-commit": true, "post-merge": false, "post-checkout": false}
	for i, h := range status.Hooks {
		if h.Name != wantOrder[i] {
			t.Errorf("Status.Hooks[%d].Name = %q, want %q", i, h.Name, wantOrder[i])
		}
		if h.Installed != wantInstalled[h.Name] {
			t.Errorf("Status.Hooks[%s].Installed = %v, want %v", h.Name, h.Installed, wantInstalled[h.Name])
		}
	}
}

// TestStatus_MarkerPresentButNotExecutable_ReportsExecutableFalse is the
// IN-03 regression test: fsatomic.WriteFile's atomic rename and the
// subsequent best-effort os.Chmod(0755) are two separate steps, so a hook
// file can end up with the marker text present but the exec bit unset
// (e.g. an external `chmod -x`, or a crash between the two steps). Status
// must report Installed=true (marker present, TS-parity check) but
// Executable=false (Go-only addition) rather than silently claiming full
// health.
func TestStatus_MarkerPresentButNotExecutable_ReportsExecutableFalse(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	installedContent := "#!/bin/sh\n" + markerBlock() + "\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(installedContent), 0o644); err != nil { // no exec bit
		t.Fatalf("WriteFile post-commit: %v", err)
	}

	status := Status(context.Background(), root)

	found := false
	for _, h := range status.Hooks {
		if h.Name != "post-commit" {
			continue
		}
		found = true
		if !h.Installed {
			t.Errorf("post-commit Installed = false, want true (marker text is present)")
		}
		if h.Executable {
			t.Errorf("post-commit Executable = true, want false (no exec bit)")
		}
	}
	if !found {
		t.Fatalf("Status.Hooks missing post-commit entry: %v", status.Hooks)
	}
}

// TestStatus_MarkerTextEmbeddedInLine_ReportsNotInstalled is the WR-01
// regression test, mirroring TestRemove_MarkerTextEmbeddedInLine_NotReportedRemoved:
// marker text merely embedded inside an unrelated line (e.g. an echoed
// string) is a raw substring match but not an exact-trimmed-line match.
// Status must not report the hook as Installed in that case — the same
// "installed" signal Remove already refuses to report for this exact
// fixture (IN-04), so the two subcommands stay consistent.
func TestStatus_MarkerTextEmbeddedInLine_ReportsNotInstalled(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	embedded := "#!/bin/sh\n" + `echo "not a real ` + markerBegin + ` marker"` + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(embedded), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	status := Status(context.Background(), root)

	for _, h := range status.Hooks {
		if h.Name != "post-commit" {
			continue
		}
		if h.Installed {
			t.Fatalf("post-commit Installed = true, want false (marker text only embedded, not an exact line match)")
		}
	}
}

func TestStatus_NonRepo_ReturnsSkipped(t *testing.T) {
	root := t.TempDir()

	status := Status(context.Background(), root)

	if status.Skipped != "not a git repository" {
		t.Fatalf("Skipped = %q, want %q", status.Skipped, "not a git repository")
	}
}

// TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched is the CR-01
// end-to-end regression test: a hook file with a begin marker but no
// matching end marker (a plausible hand-edit) must not have its trailing
// content silently destroyed by Remove, and must not be reported as
// removed since nothing was actually stripped.
func TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"",
		markerBegin,
		"... (block body, end marker line deleted) ...",
		"echo after",
		"echo more-user-content",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(hooksDir, "post-commit"), []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be reported as removed (unterminated marker, strip untrusted): %v", result.Removed)
		}
	}
	got, err := os.ReadFile(filepath.Join(hooksDir, "post-commit"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != malformed {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), malformed)
	}
}

func TestRemove_NonRepo_ReturnsSkipped(t *testing.T) {
	root := t.TempDir()

	result := Remove(context.Background(), root)

	if result.Skipped != "not a git repository" {
		t.Fatalf("Skipped = %q, want %q", result.Skipped, "not a git repository")
	}
}

// TestStripMarkerBlock_NestedBegin_ReturnsUnchanged and
// TestStripMarkerBlock_DanglingEnd_ReturnsUnchanged are the CR-01
// iteration-2 regression tests: a second begin marker encountered while one
// is already open, or an end marker with no open begin, must also report
// ok=false — not just the original unterminated-single-begin case. Without
// this, a file carrying a dangling begin marker followed later by a
// well-formed begin/end pair would falsely report ok=true and silently
// drop everything between the dangling begin and the new end marker (the
// exact defect this iteration closes; see TestInstall_Install_TwiceOnMalformedFile_UserContentSurvivesBothCalls
// for the end-to-end reproduction via Install).
func TestStripMarkerBlock_NestedBegin_ReturnsUnchanged(t *testing.T) {
	content := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		markerBegin, // dangling — never closed
		"... (end marker missing) ...",
		"echo after",
		"echo more-user-content",
		"",
		markerBegin, // a second, well-formed block appended later
		"fresh block body",
		markerEnd,
	}, "\n")

	got, ok := stripMarkerBlock(content)
	if ok {
		t.Fatalf("stripMarkerBlock(nested begin) ok = true, want false")
	}
	if got != content {
		t.Fatalf("stripMarkerBlock(nested begin) = %q, want unchanged %q", got, content)
	}
}

func TestStripMarkerBlock_DanglingEnd_ReturnsUnchanged(t *testing.T) {
	content := strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		markerEnd, // no open begin
		"echo after",
	}, "\n")

	got, ok := stripMarkerBlock(content)
	if ok {
		t.Fatalf("stripMarkerBlock(dangling end) ok = true, want false")
	}
	if got != content {
		t.Fatalf("stripMarkerBlock(dangling end) = %q, want unchanged %q", got, content)
	}
}

// malformedHookFixture returns a hook file body with an unterminated begin
// marker followed by real user content — the fixture CR-01's emergent
// defect (iteration-2 review) was reproduced against: a naive recovery
// strategy that appends a fresh well-formed block after this raw content
// creates a file shape where a LATER strip can misread the new block's end
// marker as closing the dangling begin, silently eating everything in
// between.
func malformedHookFixture() string {
	return strings.Join([]string{
		"#!/bin/sh",
		"echo before",
		"",
		markerBegin,
		"... (block body, end marker line deleted) ...",
		"echo after",
		"echo more-user-content",
	}, "\n") + "\n"
}

// TestInstall_MalformedMarkerBlock_SkipsHookAndLeavesFileUntouched is the
// CR-01 iteration-2 regression test for Install's own first encounter with
// a malformed hook file: rather than falling back to appending a fresh
// block after the untrustworthy raw content (the iteration-1 recovery
// strategy that reintroduced the data-loss hazard one round-trip later),
// Install must skip the hook entirely, accumulate an error naming it, and
// leave the file byte-for-byte untouched.
func TestInstall_MalformedMarkerBlock_SkipsHookAndLeavesFileUntouched(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := malformedHookFixture()
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Install(context.Background(), root)

	for _, h := range result.Installed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Installed (malformed marker block): %v", result.Installed)
		}
	}
	foundErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "post-commit") {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatalf("Errors = %v, want an entry naming post-commit", result.Errors)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != malformed {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), malformed)
	}
}

// TestInstall_Install_TwiceOnMalformedFile_UserContentSurvivesBothCalls is
// the exact two-call reproduction from the iteration-2 review's CR-01
// finding: Install called twice in a row against a hand-damaged hook file
// must never lose "echo after"/"echo more-user-content" on the second
// call. The iteration-1 fix protected the FIRST call but, by appending a
// fresh well-formed block after the still-malformed raw content, created a
// file shape where the second Install's strip would misread the new
// block's end marker as closing the old dangling begin — silently
// deleting everything between them. With the iteration-2 fix, Install
// never writes into a file it can't trust the strip of, so the file stays
// identically malformed (and identically unmodified) across any number of
// repeated Install calls.
func TestInstall_Install_TwiceOnMalformedFile_UserContentSurvivesBothCalls(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := malformedHookFixture()
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	first := Install(context.Background(), root)
	afterFirst, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after first Install: %v", err)
	}
	if string(afterFirst) != malformed {
		t.Fatalf("after first Install, post-commit = %q, want unchanged %q", string(afterFirst), malformed)
	}
	for _, h := range first.Installed {
		if h == "post-commit" {
			t.Fatalf("first Install: post-commit should not be in Installed: %v", first.Installed)
		}
	}

	second := Install(context.Background(), root)
	afterSecond, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile after second Install: %v", err)
	}
	if !strings.Contains(string(afterSecond), "echo after") || !strings.Contains(string(afterSecond), "echo more-user-content") {
		t.Fatalf("second Install silently dropped user content: %q", string(afterSecond))
	}
	if string(afterSecond) != malformed {
		t.Fatalf("after second Install, post-commit = %q, want unchanged %q", string(afterSecond), malformed)
	}
	for _, h := range second.Installed {
		if h == "post-commit" {
			t.Fatalf("second Install: post-commit should not be in Installed: %v", second.Installed)
		}
	}
}

// TestInstall_ThenRemoveOnMalformedFile_UserContentSurvives is the
// Install-then-Remove variant of the same iteration-2 CR-01 reproduction:
// an Install call that (correctly) skips a malformed hook file must not
// leave the file in a state where a SUBSEQUENT Remove call misreads it and
// truncates user content. Since Install now never writes into an
// untrustworthy file, the file handed to Remove is identical to the
// original malformed fixture, and Remove's own existing ok==false handling
// (verified separately by TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched)
// applies unchanged.
func TestInstall_ThenRemoveOnMalformedFile_UserContentSurvives(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := malformedHookFixture()
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	Install(context.Background(), root)
	removeResult := Remove(context.Background(), root)

	for _, h := range removeResult.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be reported as removed (malformed marker block): %v", removeResult.Removed)
		}
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(got), "echo after") || !strings.Contains(string(got), "echo more-user-content") {
		t.Fatalf("Install->Remove silently dropped user content: %q", string(got))
	}
	if string(got) != malformed {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), malformed)
	}
}

// TestInstall_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError
// is the CR-02 (iteration-3) regression test: an existing hook file that
// exists but can't be read (permission bit revoked) must not be silently
// treated as "file absent" and overwritten with a fresh seed block. Install
// must skip the hook, accumulate an error naming it, and leave the file's
// content byte-for-byte untouched — reproducing the review's exact repro
// (chmod 0000 a hook seeded with sentinel content, then Install). Skipped
// under root and on Windows, matching the existing skip convention for
// permission-based tests in this file.
func TestInstall_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	sentinel := "#!/bin/sh\necho SUPER-IMPORTANT-USER-CONTENT\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(sentinel), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(file, 0o000); err != nil {
		t.Fatalf("Chmod(%s, 0000): %v", file, err)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o644) })

	result := Install(context.Background(), root)

	for _, h := range result.Installed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Installed (unreadable existing file): %v", result.Installed)
		}
	}
	foundErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "post-commit") {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatalf("Errors = %v, want an entry naming post-commit", result.Errors)
	}

	// Restore read permission so the test itself can verify content
	// survived — the assertion under test is that Install never destroyed
	// it, not that the file stays permanently unreadable.
	if err := os.Chmod(file, 0o644); err != nil {
		t.Fatalf("Chmod(%s, 0644) for verification: %v", file, err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != sentinel {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), sentinel)
	}
}

// TestRemove_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError
// is the WR-02 regression test, mirroring
// TestInstall_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError:
// an installed hook file whose read permission is revoked must not be
// silently skipped with zero signal. Remove must skip the hook (write
// nothing, leave content untouched — the existing, correct half of the
// behavior) but now also accumulate an error naming it in
// RemoveResult.Errors, mirroring Install's CR-02 handling of the identical
// read failure. Skipped under root and on Windows, matching the existing
// skip convention for permission-based tests in this file.
func TestRemove_UnreadableExistingFile_LeavesFileUntouchedAndAccumulatesError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits don't apply on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	installedContent := "#!/bin/sh\n" + markerBlock() + "\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(installedContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(file, 0o000); err != nil {
		t.Fatalf("Chmod(%s, 0000): %v", file, err)
	}
	t.Cleanup(func() { _ = os.Chmod(file, 0o644) })

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Removed (unreadable existing file): %v", result.Removed)
		}
	}
	foundErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "post-commit") {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatalf("Errors = %v, want an entry naming post-commit", result.Errors)
	}

	// Restore read permission so the test itself can verify content
	// survived — the assertion under test is that Remove never destroyed
	// it, not that the file stays permanently unreadable.
	if err := os.Chmod(file, 0o644); err != nil {
		t.Fatalf("Chmod(%s, 0644) for verification: %v", file, err)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != installedContent {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), installedContent)
	}
}

// TestRemove_DanglingEndMarkerOnly_NotReportedRemoved is the WR-05
// regression test: a hand-edited hook file containing only a dangling end
// marker (no begin marker anywhere) must not be silently treated as "never
// installed, nothing to do" — stripMarkerBlock correctly flags this shape
// as malformed (ok=false), and Remove must honor that the same way Install
// does, rather than reporting it removed or leaving inconsistent signals
// between the two subcommands.
func TestRemove_DanglingEndMarkerOnly_NotReportedRemoved(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	danglingEndOnly := "echo hi\n" + markerEnd + "\necho bye\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(danglingEndOnly), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Removed (dangling end marker only, malformed): %v", result.Removed)
		}
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != danglingEndOnly {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), danglingEndOnly)
	}
}

// TestRemove_MalformedMarkerBlock_LeavesFileUntouchedAndAccumulatesError is
// the round-6 WR-01 regression test: Remove's ok==false branch must mirror
// Install's CR-01 handling of the identical condition — not just leave a
// malformed hook file untouched (the existing, correct half of the
// behavior verified by TestRemove_UnterminatedMarkerBlock_LeavesFileUntouched)
// but also accumulate an actionable error naming the hook in
// RemoveResult.Errors, so callers don't silently report success against a
// hand-damaged hook file.
func TestRemove_MalformedMarkerBlock_LeavesFileUntouchedAndAccumulatesError(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	malformed := malformedHookFixture()
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(malformed), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Removed (malformed marker block): %v", result.Removed)
		}
	}
	foundErr := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "post-commit") {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatalf("Errors = %v, want an entry naming post-commit", result.Errors)
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != malformed {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), malformed)
	}
}

// TestRemove_MarkerTextEmbeddedInLine_NotReportedRemoved is the IN-04
// regression test: marker text embedded inside an unrelated line (e.g. an
// echoed string) is a raw substring match but not an exact-trimmed-line
// match, so stripMarkerBlock strips nothing. Remove must not report the
// hook as Removed or rewrite the file in that case — "reported as removed"
// must imply "content actually changed."
func TestRemove_MarkerTextEmbeddedInLine_NotReportedRemoved(t *testing.T) {
	root := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	embedded := "#!/bin/sh\n" + `echo "not a real ` + markerBegin + ` marker"` + "\n"
	file := filepath.Join(hooksDir, "post-commit")
	if err := os.WriteFile(file, []byte(embedded), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	result := Remove(context.Background(), root)

	for _, h := range result.Removed {
		if h == "post-commit" {
			t.Fatalf("post-commit should not be in Removed (marker text only embedded, not an exact line match): %v", result.Removed)
		}
	}
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != embedded {
		t.Fatalf("post-commit content changed = %q, want unchanged %q", string(got), embedded)
	}
}
