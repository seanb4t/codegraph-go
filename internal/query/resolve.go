// Package query is the read-only engine over a frozen internal/graphstore
// GraphStore (D-02): one Snapshot per invocation, never a write path. CLI
// commands and the MCP server share this single engine (D-08b) so their
// output shapes cannot drift into two code paths.
package query

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/gitmeta"
	"github.com/seanb4t/codegraph-go/internal/indexdir"
)

// ErrNotInitialized mirrors the internal/cli sentinel of the same name
// (see internal/cli/root.go) but is declared locally so internal/query has
// no reason to import internal/cli (which would invert the intended
// CLI-depends-on-query dependency direction). Callers compare against this
// sentinel via errors.Is/errors.As, not string matching.
var ErrNotInitialized = errors.New("query: not initialized")

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
// worktree's index: a worktree outside the main checkout never
// passes through it on the way up. The walk notes the nearest .git entry,
// and only when that entry is a file (a linked worktree or a submodule)
// is git asked for the main worktree, so an ordinary repository without an
// index never starts a git process. The repository must record the
// worktree both ways (gitmeta.LinkedWorktreeMain), so a hand-written .git
// file cannot point the fallback at an unrelated repository's index. Any
// git failure, an unrecorded or bare worktree, or a main worktree that is
// the home directory or holds no .codegraph returns ErrNotInitialized.
//
// A .codegraph that is a directory or a symlink, at the walk's directory
// or at the main worktree, must resolve inside the directory that holds
// it, and so must its store (indexdir.Resolve). Otherwise ResolveIndex returns
// an *indexdir.OutsideRootError and does not walk further. A symlink whose
// missing target is inside that directory is walked past.
func ResolveIndex(start string) (Resolution, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return Resolution{}, err
	}
	home := homeDir()
	worktree := ""
	sawGit := false
	for {
		if dir != home {
			found, err := indexAt(dir)
			if err != nil {
				return Resolution{}, err
			}
			if found {
				return Resolution{IndexRoot: dir, ConfineRoot: dir}, nil
			}
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
		main, err := mainWorktreeIndex(worktree, home)
		if err != nil {
			return Resolution{}, err
		}
		if main != "" {
			return Resolution{IndexRoot: main, ConfineRoot: worktree}, nil
		}
	}
	return Resolution{}, ErrNotInitialized
}

// ProjectIndexed reports whether a hook running in project directory dir
// should treat it as indexed: dir holds .codegraph, or dir is the root of
// a linked worktree whose main worktree holds one and is not the home
// directory. An index that ResolveIndex would refuse does not count.
// Unlike ResolveIndex it does not walk upward; the hooks check the project
// directory only (NUDGE-01, D-22).
func ProjectIndexed(dir string) bool {
	found, err := indexAt(dir)
	if err != nil {
		return false
	}
	if found {
		return true
	}
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	main, err := mainWorktreeIndex(dir, homeDir())
	return err == nil && main != ""
}

// indexAt reports whether dir holds a .codegraph directory. When dir's
// .codegraph is a directory or a symlink, it first returns indexdir.Resolve's
// error, if any.
func indexAt(dir string) (bool, error) {
	index := filepath.Join(dir, indexdir.Name)
	info, err := os.Lstat(index)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() && info.Mode()&fs.ModeSymlink == 0 {
		return false, nil
	}
	if _, err := indexdir.Resolve(dir); err != nil {
		return false, err
	}
	return isDir(index), nil
}

// mainWorktreeIndex returns the main worktree of the repository that
// worktree is a recorded linked worktree of, when it holds a .codegraph
// directory and is not the home directory, and "" otherwise. It returns
// indexAt's error for the main worktree. LinkedWorktreeMain resolves
// symlinks, so the home directory is compared both as given and resolved.
func mainWorktreeIndex(worktree, home string) (string, error) {
	main := gitmeta.LinkedWorktreeMain(context.Background(), worktree)
	if main == "" {
		return "", nil
	}
	if home != "" {
		if main == home {
			return "", nil
		}
		if resolved, err := filepath.EvalSymlinks(home); err == nil && main == resolved {
			return "", nil
		}
	}
	found, err := indexAt(main)
	if err != nil || !found {
		return "", err
	}
	return main, nil
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
