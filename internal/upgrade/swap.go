package upgrade

import (
	"fmt"
	"os"
	"path/filepath"
)

// checkWritable probes whether the current user can replace targetPath, by
// creating and immediately removing a temp file in its directory — the
// same directory-write operation atomicSwap performs for real. Used both
// as atomicSwap's own precondition and by upgrade.Run to fail fast BEFORE
// downloading anything (D-13: no wasted download if the install can't be
// replaced). Deliberately does NOT open targetPath itself for writing: a
// currently-executing binary can reject a direct write-open on some
// platforms even though replacing it via temp-file+rename is exactly what
// atomicSwap is designed to do safely.
func checkWritable(targetPath string) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".codegraph-upgrade-writable-check-*")
	if err != nil {
		return fmt.Errorf("upgrade: %s is not writable by the current user; refusing to upgrade (no changes made): %w", targetPath, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}

// atomicSwap replaces the binary at targetPath with newBinary using a
// temp-file-in-the-same-directory + rename dance (D-13). The temp file is
// fully written and made executable BEFORE anything touches targetPath, so
// a crash/interrupt at any point before the final rename leaves the
// ORIGINAL binary intact (T-06-06-03) — os.Rename within one directory is
// atomic on POSIX, and POSIX is the whole supported surface: native
// Windows is not a supported platform (quick task 260807-gho), so the
// rename-self-aside dance a running .exe would have required is gone
// (pattern source: minio/selfupdate's apply.go, referenced per RESEARCH —
// not a dependency, D-13/Alternatives Considered).
func atomicSwap(targetPath string, newBinary []byte) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		return fmt.Errorf("upgrade: stat target: %w", err)
	}

	if err := checkWritable(targetPath); err != nil {
		return err
	}

	dir := filepath.Dir(targetPath)
	tmp, err := os.CreateTemp(dir, ".codegraph-upgrade-*")
	if err != nil {
		return fmt.Errorf("upgrade: %s is not writable by the current user; refusing to upgrade (no changes made): %w", targetPath, err)
	}
	tmpPath := tmp.Name()

	// The temp file is the only thing that can be left behind on an
	// early-return error below — cleanupTemp stays true until the final
	// rename succeeds, at which point tmpPath no longer exists under its
	// own name (T-06-06-03: no partial state on interruption).
	cleanupTemp := true
	defer func() {
		if cleanupTemp {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(newBinary); err != nil {
		tmp.Close()
		return fmt.Errorf("upgrade: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("upgrade: close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, info.Mode().Perm()|0o111); err != nil {
		return fmt.Errorf("upgrade: chmod temp file: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("upgrade: rename new binary into place: %w", err)
	}
	cleanupTemp = false
	return nil
}
