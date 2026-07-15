package gitmeta

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// runGit runs `git <args...>` in dir with a fixed set of deterministic,
// hermetic flags prepended so fixture commits are reproducible across
// machines and CI (D-15). Any failure — including git being absent from
// PATH entirely — skips the calling test rather than failing the suite:
// git being unavailable degrades every gitmeta call to "no signal" per
// WORK-03, and the same philosophy applies to the fixtures that exercise it.
//
// protocol.file.allow=always is mandatory: modern git refuses `submodule
// add` from a local file:// path without it, and its absence would silently
// skip the submodule fixture (Test 3) rather than exercising gate 4 at all.
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
// the repo has a real HEAD (worktree/submodule operations below all need
// at least one commit to branch from). Returns dir for chaining.
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

// newLinkedWorktreeFixture builds a main checkout plus one linked worktree
// via `git worktree add`, sitting OUTSIDE the main checkout. This is the
// textbook true-positive shape: start=the worktree, index=the main tree ⇒
// MISMATCH (D-15).
func newLinkedWorktreeFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	main := initRepo(t, filepath.Join(tmp, "main"))
	wt := filepath.Join(tmp, "wt")
	runGit(t, main, "worktree", "add", "-b", "feature", wt)
	return wt, main
}

// newClaudeWorktreeFixture builds a linked worktree NESTED INSIDE the main
// checkout at .claude/worktrees/<name>/ — the motivating GSD layout (see
// CONTEXT.md § Specific Ideas): a linked worktree placed under a
// .gitignore'd path inside the parent tree, exactly what the .codegraph/
// upward walk resolves wrong. start=the nested worktree, index=the main
// tree ⇒ MISMATCH (D-15).
func newClaudeWorktreeFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	main := initRepo(t, filepath.Join(tmp, "main"))
	wt := filepath.Join(main, ".claude", "worktrees", "phase-2")
	runGit(t, main, "worktree", "add", "-b", "phase-2", wt)
	return wt, main
}

// newSubmoduleFixture builds a parent repo with a registered git submodule.
// A submodule is a DIFFERENT repository from its parent, whose files the
// parent's index already covers by descending into it at index time.
//
// ★ Gate-4 polarity (D-02, CONTEXT § Specific Ideas): the submodule's git
// common dir DIFFERS from the parent's, and a DIFFERING common dir SUPPRESSES
// the warning — the inverse of what the requirement text's shorthand implies.
// A genuine borrowed worktree instead SHARES its common dir with the index
// root. Reading only "detect a mismatched worktree" naturally leads to
// "differing dirs signal a problem"; here they signal the opposite — a
// distinct, already-covered repository, not a borrowed index. Do not invert
// this in a future "simplification". start=<parent>/sub, index=<parent> ⇒
// NO MISMATCH (D-15).
func newSubmoduleFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	parent := initRepo(t, filepath.Join(tmp, "parent"))
	child := initRepo(t, filepath.Join(tmp, "child"))
	runGit(t, parent, "submodule", "add", child, "sub")
	runGit(t, parent, "commit", "-m", "add submodule")
	return filepath.Join(parent, "sub"), parent
}

// newNestedCloneFixture builds a parent repo with a plain embedded clone
// (a second `git init` inside the parent's working tree, with no gitlink
// and no submodule registration). Like a submodule, this is a genuinely
// DIFFERENT repository the parent's index already covers when it descends
// into the directory during indexing.
//
// ★ Same gate-4 polarity as newSubmoduleFixture above: the embedded clone's
// git common dir differs from the parent's, and that DIFFERENCE suppresses
// the warning rather than triggering it — a different repo, not a borrowed
// worktree of the same one. start=<parent>/embedded, index=<parent> ⇒
// NO MISMATCH (D-15).
func newNestedCloneFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	parent := initRepo(t, filepath.Join(tmp, "parent"))
	embedded := initRepo(t, filepath.Join(parent, "embedded"))
	return embedded, parent
}

// newMonorepoSubdirFixture builds a single repo with a plain subdirectory
// (no worktree, no submodule — just `mkdir`). start=that subdir,
// index=the repo root itself ⇒ NO MISMATCH: the index root IS start's own
// worktree root, so gate 2 short-circuits before gate 3 is even reached
// (D-15). See TestFixtureVerdicts' "monorepo-subdir-plain-ancestor" subtest
// for the companion case that isolates gate 3 specifically: an index root
// that is neither start's own worktree root NOR itself a worktree root at
// all (a plain non-git ancestor directory).
func newMonorepoSubdirFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	repo := initRepo(t, filepath.Join(tmp, "repo"))
	subdir := filepath.Join(repo, "services", "api")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", subdir, err)
	}
	return subdir, repo
}

// newSymlinkedFixture builds one repo and a symlink pointing at its root.
// start=the symlink path, index=the real repo root ⇒ NO MISMATCH once both
// sides are resolved through EvalSymlinks to the same tree (gate 2, D-15).
// macOS's own t.TempDir() already exercises the /var -> /private/var
// symlink incidentally; this fixture adds an explicit, intentional symlink
// on top of that.
func newSymlinkedFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	tmp := t.TempDir()
	repo := initRepo(t, filepath.Join(tmp, "repo"))
	link := filepath.Join(tmp, "repo-link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("os.Symlink unsupported on this machine: %v", err)
	}
	return link, repo
}

// newNonGitFixture returns two bare temp directories with no git
// initialization at all — WORK-03's "not a repo" degradation case.
// start and index are both outside any git working tree ⇒ NO MISMATCH
// (gate 1: gitWorktreeRoot(start) is empty, D-15).
func newNonGitFixture(t *testing.T) (startPath, indexRoot string) {
	t.Helper()
	return t.TempDir(), t.TempDir()
}
