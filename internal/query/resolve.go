// Package query is the read-only engine over a frozen internal/graphstore
// GraphStore (D-02): one Snapshot per invocation, never a write path. CLI
// commands and the MCP server share this single engine (D-08b) so their
// output shapes cannot drift into two code paths.
package query

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
)

// ErrNotInitialized mirrors the internal/cli sentinel of the same name
// (see internal/cli/root.go) but is declared locally so internal/query has
// no reason to import internal/cli (which would invert the intended
// CLI-depends-on-query dependency direction). Callers compare against this
// sentinel via errors.Is/errors.As, not string matching.
var ErrNotInitialized = errors.New("query: not initialized")

// codegraphDirName is the directory ResolveCodegraphDir looks for at each
// level of the upward walk — kept in lockstep with internal/cli's constant
// of the same name and value (D-01b layout).
const codegraphDirName = ".codegraph"

// Resolution is where an index was found for a start directory.
// IndexRoot holds the .codegraph directory. ConfineRoot is the tree a
// caller that started there may read: IndexRoot itself when the upward walk
// found the index, or the linked worktree's root when the index came from
// its main worktree. ConfineRoot is always the start directory or one of
// its ancestors.
type Resolution struct {
	IndexRoot   string
	ConfineRoot string
}

// ResolveCodegraphDir returns ResolveIndex(start).IndexRoot.
func ResolveCodegraphDir(start string) (string, error) {
	r, err := ResolveIndex(start)
	return r.IndexRoot, err
}

// ResolveIndex walks filepath.Dir upward from start (D-01a, RESEARCH
// Pattern 4), returning the nearest ancestor directory (inclusive of start
// itself) that contains a .codegraph subdirectory. start is resolved to an
// absolute path first via filepath.Abs (V5 — no unbounded/relative path is
// ever stat'd against unexpected cwd state).
//
// The user's home directory is never an index root: a .codegraph there is
// skipped and the walk continues to its parent. Older releases kept
// per-user state in ~/.codegraph, and anything else may still create a
// directory of that name, so without the skip every unindexed directory
// under the home directory would resolve to the home directory itself.
// The comparison is on filepath.Clean of both paths, with no
// symlink evaluation; when the home directory cannot be determined,
// nothing is skipped.
//
// When the walk finds nothing, a linked worktree falls back to its main
// worktree's index (#139): a worktree outside the main checkout never
// passes through it on the way up. The walk notes the nearest .git entry,
// and only when that entry is a file (a linked worktree or a submodule)
// is git asked for the main worktree, so an ordinary repository without an
// index never starts a git process. Any git failure, a bare repository,
// or a main worktree that is the home directory or holds no .codegraph
// returns ErrNotInitialized.
func ResolveIndex(start string) (Resolution, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return Resolution{}, err
	}
	home := homeDir()
	worktree := ""
	sawGit := false
	for {
		if dir != home && isDir(filepath.Join(dir, codegraphDirName)) {
			return Resolution{IndexRoot: dir, ConfineRoot: dir}, nil
		}
		if !sawGit {
			if info, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
				sawGit = true
				if info.Mode().IsRegular() {
					worktree = dir
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if worktree != "" {
		if main := mainWorktreeIndex(worktree, home); main != "" {
			return Resolution{IndexRoot: main, ConfineRoot: worktree}, nil
		}
	}
	return Resolution{}, ErrNotInitialized
}

// ProjectIndexed reports whether a hook running in project directory dir
// should treat it as indexed: dir holds .codegraph, or dir is the root of
// a linked worktree whose main worktree holds one and is not the home
// directory. Unlike ResolveIndex it does not walk upward; the hooks check
// the project directory only (NUDGE-01, D-22).
func ProjectIndexed(dir string) bool {
	if isDir(filepath.Join(dir, codegraphDirName)) {
		return true
	}
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return mainWorktreeIndex(dir, homeDir()) != ""
}

// mainWorktreeIndex returns the main worktree of the repository worktree
// belongs to when it holds a .codegraph directory and is not the home
// directory, and "" otherwise. MainWorktree resolves symlinks, so the home
// directory is compared both as given and resolved.
func mainWorktreeIndex(worktree, home string) string {
	main := gitmeta.MainWorktree(context.Background(), worktree)
	if main == "" || !isDir(filepath.Join(main, codegraphDirName)) {
		return ""
	}
	if home != "" {
		if main == home {
			return ""
		}
		if resolved, err := filepath.EvalSymlinks(home); err == nil && main == resolved {
			return ""
		}
	}
	return main
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Clean(h)
	}
	return ""
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
