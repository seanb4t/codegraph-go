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
	"sync"
	"time"
)

// gitTimeout bounds every git subprocess this package spawns, 5000ms
// (D-03): these calls run on long-lived, latency-sensitive paths (a
// status/query call, an MCP tool handler), where an unbounded git hang
// would otherwise trip the daemon's 60s liveness watchdog and take down
// an otherwise-healthy process (tracked as issue #1139).
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

// MainWorktree returns the absolute, symlink-resolved root of the main
// worktree of the repository dir belongs to, or "" when dir isn't in a
// repository, the repository is bare (it has no main worktree), or git is
// unavailable or slow. `git worktree list --porcelain` lists the main
// worktree first from every worktree of a repository, so a linked worktree
// anywhere on disk finds it, and a main checkout made with
// --separate-git-dir (whose common dir is not <main>/.git) is still right.
//
// An answer from a git run that exited 0, including "bare", is remembered
// for the life of the process, keyed by filepath.Clean(dir): the MCP server
// resolves its index on every request, and one git spawn costs tens of
// milliseconds. A failure is not remembered, so the next call tries again.
func MainWorktree(ctx context.Context, dir string) string {
	key := filepath.Clean(dir)
	if main, ok := mainWorktrees.get(key); ok {
		return main
	}
	main, ok := mainWorktree(ctx, dir)
	if ok {
		mainWorktrees.put(key, main)
	}
	return main
}

// mainWorktree runs the uncached lookup. ok reports whether git ran and
// exited 0, which is what makes the answer safe to cache.
func mainWorktree(ctx context.Context, dir string) (main string, ok bool) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	cmd.Dir = dir
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	// The first record runs to the first blank line.
	for line := range strings.SplitSeq(string(out), "\n") {
		switch {
		case line == "":
			return realpathOrEmpty(main), true
		case line == "bare":
			return "", true
		case main == "" && strings.HasPrefix(line, "worktree "):
			main = strings.TrimPrefix(line, "worktree ")
		}
	}
	return realpathOrEmpty(main), true
}

func realpathOrEmpty(p string) string {
	if p == "" {
		return ""
	}
	return realpath(p)
}

// mainWorktrees is MainWorktree's process-lifetime cache, bounded like
// CachingDetector's (maxCacheEntries, WR-02): a client-supplied MCP "path"
// can mint new keys, so a full cache is dropped and refilled.
var mainWorktrees = &mainWorktreeCache{entries: make(map[string]string)}

type mainWorktreeCache struct {
	mu      sync.Mutex
	entries map[string]string
}

func (c *mainWorktreeCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.entries[key]
	return v, ok
}

func (c *mainWorktreeCache) put(key, main string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= maxCacheEntries {
		c.entries = make(map[string]string)
	}
	c.entries[key] = main
}

// realpath resolves p to an absolute path, then attempts to resolve
// symlinks. On any error (p doesn't exist, permission denied, etc.) fall
// back to the plain absolute path rather than propagating the error.
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
