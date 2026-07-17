// Package fsatomic provides a single crash-safe file-write primitive
// shared by internal/agents (external agent configs) and internal/githooks
// (git hook scripts).
//
// D-09 narrowing: this package extracts ONLY the atomic-write mechanism
// (temp file in the target directory followed by os.Rename) out of
// internal/agents/shared.go's atomicWriteFile. The marker-splice helpers
// (replaceOrAppendMarkedSection / removeMarkedSection) are deliberately
// NOT extracted here — internal/agents' in-place HTML-marker replacement
// and internal/githooks' trimmed-line strip-then-re-append-at-end
// semantics are different enough that a shared abstraction would distort
// one side or the other. Two small correct implementations beat one wrong
// shared one.
package fsatomic

import (
	"os"
	"path/filepath"
)

// WriteFile writes content to path via a temp file created in the same
// directory followed by os.Rename, so a crash or interrupt mid-write
// never leaves a truncated or corrupted file on disk. Callers across
// internal/agents and internal/githooks funnel every hook/config write
// through this one function.
//
// os.CreateTemp creates the temp file with mode 0600 on POSIX; if path
// already exists, its mode is preserved by chmod'ing the temp file to
// match before the rename — otherwise the first write to a file that
// previously had, say, 0644 permissions would silently tighten it to
// 0600. A new file gets the conventional 0644 default.
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".codegraph-write-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
