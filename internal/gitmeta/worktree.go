// Package gitmeta is the stdlib-only, best-effort git introspection layer
// that detects when a resolved CodeGraph index belongs to a different git
// working tree than the caller (WORK-01/02/03). It shells out to the local
// `git` binary via os/exec — no pure-Go git library, no CGo (D-03/D-04) —
// and is deliberately free of internal/query and internal/mcp concerns, so
// Phase 5's git sync hooks can reuse it unchanged.
//
// Every function here degrades to a safe zero value on ANY failure: missing
// git, a non-repo path, a timeout, or a transient error all report "no
// signal" rather than an error. A read query must never fail or block on
// git being unavailable, slow, or absent (WORK-03).
package gitmeta

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitTimeout bounds every git subprocess this package spawns. Ported from
// TS sync/worktree.js's `timeout: 5000` (D-03): these calls run on
// long-lived, latency-sensitive paths (a status/query call, an MCP tool
// handler), where an unbounded git hang would otherwise trip the daemon's
// 60s liveness watchdog and take down an otherwise-healthy process (TS
// issue #1139).
const gitTimeout = 5 * time.Second

// WorktreeRoot returns the absolute, symlink-resolved toplevel of the git
// working tree that dir belongs to, or "" when dir isn't inside a git repo
// (or git is unavailable/slow). `git rev-parse --show-toplevel` reports the
// PER-WORKTREE root: the main checkout and each linked worktree resolve to
// their own distinct directory — exactly the distinction detection needs.
func WorktreeRoot(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	return realpath(trimmed)
}

// CommonDir returns the absolute, symlink-resolved git COMMON directory for
// dir — the shared `.git` every worktree of one repository points at, or ""
// when dir isn't a repo. Linked worktrees of the SAME repository report the
// SAME common dir; a submodule or an embedded clone is a DIFFERENT
// repository and reports its own (e.g. `.git/modules/<name>`, or its own
// `.git`). That distinction is what separates a genuine borrowed worktree
// from a nested repo the parent index already covers (see DetectIndexMismatch
// gate 4).
func CommonDir(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-common-dir")
	cmd.Dir = dir
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return ""
	}
	// --git-common-dir is relative to cwd unless already absolute (D-03).
	// Skipping this makes gate 4 compare a bare ".git" string against an
	// absolute path and mis-fire.
	resolved := trimmed
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(dir, resolved)
	}
	return realpath(resolved)
}

// realpath mirrors TS's realpathSync-with-fallback: resolve p to an
// absolute path, then attempt to resolve symlinks. On any error (p doesn't
// exist, permission denied, etc.) fall back to the plain absolute path
// rather than propagating the error (TS's `catch { return path.resolve(p) }`).
func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}
