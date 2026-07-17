package gitmeta

import (
	"context"
	"os/exec"
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
