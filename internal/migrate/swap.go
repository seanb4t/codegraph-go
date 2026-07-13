package migrate

import (
	"fmt"
	"os"
	"path/filepath"
)

// checkWritableDir probes whether the current user can write into
// parentDir, by creating and immediately removing a temp file there — the
// same directory-write operation atomicSwapDir performs for real. Used to
// fail fast BEFORE running the whole migration (mirrors
// internal/upgrade/swap.go's checkWritable "fail fast before downloading
// anything" rationale).
func checkWritableDir(parentDir string) error {
	f, err := os.CreateTemp(parentDir, ".codegraph-migrate-writable-check-*")
	if err != nil {
		return fmt.Errorf("migrate: %s is not writable by the current user; refusing to migrate (no changes made): %w", parentDir, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

// siblingTempDir creates a fresh directory in filepath.Dir(targetDir) — the
// SAME parent as the final target, never os.TempDir() — so the later
// atomicSwapDir rename is a same-filesystem atomic rename and never fails
// with EXDEV (RESEARCH Pitfall 1).
func siblingTempDir(targetDir string) (string, error) {
	parent := filepath.Dir(targetDir)
	tmp, err := os.MkdirTemp(parent, ".codegraph.migrate-tmp-*")
	if err != nil {
		return "", fmt.Errorf("migrate: create sibling temp dir: %w", err)
	}
	return tmp, nil
}

// atomicSwapDir replaces the directory at targetDir with tmpDir's contents
// using a three-step rename dance (D-07), extending
// internal/upgrade/swap.go's single-file temp+rename discipline to a
// directory target (os.Rename onto an existing non-empty directory fails
// on most platforms, so a plain single rename is not enough here):
//
//  1. if targetDir exists, rename it aside to targetDir+".old"
//  2. rename tmpDir into targetDir
//  3. remove targetDir+".old"
//
// If step 2 fails after step 1 has already renamed the original aside,
// atomicSwapDir attempts to restore the original from targetDir+".old"
// (WR-04 discipline, mirroring upgrade/swap.go's swapWindows): if the
// restore succeeds, only the original step-2 error is returned and the
// target is left exactly as it was before the call. If the restore ALSO
// fails, both errors are reported and the message names the .old path as
// the manual recovery location — the target must never silently end up
// pointing at nothing.
//
// A step-3 (cleanup of the now-stale .old) failure is surfaced as a
// returned, non-fatal error: the swap itself has already succeeded (the
// target holds the new contents) by the time step 3 runs, so this error
// indicates only a leftover .old directory to clean up manually, never a
// lost or corrupted target.
//
// On Windows, renaming onto an existing path is not guaranteed atomic the
// way POSIX rename is; the same temp+rename structure still bounds the
// torn-state window to the brief instant between steps 1 and 2 rather than
// eliminating it entirely.
func atomicSwapDir(tmpDir, targetDir string) error {
	asidePath := targetDir + ".old"

	targetExists := false
	if _, err := os.Stat(targetDir); err == nil {
		targetExists = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("migrate: stat target dir %s: %w", targetDir, err)
	}

	if targetExists {
		os.RemoveAll(asidePath) // best-effort: a stale .old from an interrupted prior migration
		if err := os.Rename(targetDir, asidePath); err != nil {
			return fmt.Errorf("migrate: rename existing target aside: %w", err)
		}
	}

	if err := os.Rename(tmpDir, targetDir); err != nil {
		if !targetExists {
			return fmt.Errorf("migrate: rename new store into place: %w", err)
		}
		if restoreErr := os.Rename(asidePath, targetDir); restoreErr != nil {
			return fmt.Errorf("migrate: swap failed (%v) AND restoring the original store failed (%v); "+
				"your original store is preserved at %s — rename it back to %s manually", err, restoreErr, asidePath, targetDir)
		}
		return fmt.Errorf("migrate: rename new store into place: %w", err)
	}

	if targetExists {
		if err := os.RemoveAll(asidePath); err != nil {
			return fmt.Errorf("migrate: swap succeeded but cleanup of stale %s failed (manual removal recommended): %w", asidePath, err)
		}
	}

	return nil
}
