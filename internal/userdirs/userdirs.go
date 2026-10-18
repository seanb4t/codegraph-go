// Package userdirs resolves the directories that hold codegraph's per-user
// files, following the XDG Base Directory layout on every platform: state
// under XDG_STATE_HOME (default ~/.local/state) and configuration under
// XDG_CONFIG_HOME (default ~/.config). Nothing codegraph owns lives under
// ~/.codegraph: that name is the per-repository index marker, and a
// directory of that name in the home directory would be taken for an index.
//
// The functions only resolve paths. Whoever writes a file creates the
// directory it needs.
package userdirs

import (
	"os"
	"path/filepath"
)

// appDirName is the subdirectory codegraph owns inside each XDG base
// directory.
const appDirName = "codegraph"

// StateDir returns the per-user state directory: $XDG_STATE_HOME/codegraph
// when XDG_STATE_HOME is an absolute path, otherwise
// <home>/.local/state/codegraph.
func StateDir() (string, error) {
	return resolve("XDG_STATE_HOME", ".local", "state")
}

// ConfigDir returns the per-user configuration directory:
// $XDG_CONFIG_HOME/codegraph when XDG_CONFIG_HOME is an absolute path,
// otherwise <home>/.config/codegraph.
func ConfigDir() (string, error) {
	return resolve("XDG_CONFIG_HOME", ".config")
}

// resolve applies the XDG rule for one base directory: the variable wins
// only when it is set to an absolute path (the specification says a
// relative value is invalid and must be ignored); otherwise the default
// path under the home directory applies.
func resolve(envKey string, defaultUnderHome ...string) (string, error) {
	if base := os.Getenv(envKey); filepath.IsAbs(base) {
		return filepath.Join(base, appDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	parts := append([]string{home}, defaultUnderHome...)
	parts = append(parts, appDirName)
	return filepath.Join(parts...), nil
}
