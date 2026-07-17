package gitmeta

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsGitRepo reports whether dir is inside a git working tree. It follows
// the same exec contract as WorktreeRoot/CommonDir (gitTimeout, cmd.Dir,
// cmd.Stdin nil) and degrades to false on any failure — missing git, a
// non-repo path, a timeout, or a transient error all report "no signal"
// rather than propagating an error (D-10, ported from TS sync/git-hooks.js
// isGitRepo).
func IsGitRepo(ctx context.Context, dir string) bool {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// HooksDir returns the git hooks directory for projectRoot, resolved via
// `git rev-parse --git-path hooks` — the only correct way to honor
// core.hooksPath and linked worktrees (which share the main checkout's
// common hooks dir). A relative result is joined against projectRoot; an
// absolute result (the case for linked worktrees) is passed through
// unchanged. Unlike CommonDir, this deliberately does NOT call realpath:
// D-04 specifies resolve-or-passthrough only, not symlink resolution.
// Degrades to "" on any error, empty output, or non-repo projectRoot.
func HooksDir(ctx context.Context, projectRoot string) string {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-path", "hooks")
	cmd.Dir = projectRoot
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(projectRoot, trimmed)
}
