// Package query is the read-only engine over a frozen internal/graphstore
// GraphStore (D-02): one Snapshot per invocation, never a write path. CLI
// commands and the MCP server share this single engine (D-08b) so their
// output shapes cannot drift into two code paths.
package query

import (
	"errors"
	"os"
	"path/filepath"
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

// ResolveCodegraphDir walks filepath.Dir upward from start (D-01a,
// RESEARCH Pattern 4), returning the nearest ancestor directory (inclusive
// of start itself) that contains a .codegraph subdirectory. It returns
// ErrNotInitialized once the filesystem root is reached without finding
// one. start is resolved to an absolute path first via filepath.Abs (V5 —
// no unbounded/relative path is ever stat'd against unexpected cwd state).
func ResolveCodegraphDir(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, codegraphDirName)
		info, statErr := os.Stat(candidate)
		if statErr == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInitialized
		}
		dir = parent
	}
}
