package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMarker(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "marker"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile marker in %s: %v", dir, err)
	}
}

func readMarker(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "marker"))
	if err != nil {
		t.Fatalf("ReadFile marker in %s: %v", dir, err)
	}
	return string(b)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestSiblingTempDir_DistinctSameParent proves siblingTempDir creates a
// fresh directory in filepath.Dir(target) (same parent → same filesystem)
// and that two calls yield distinct dirs.
func TestSiblingTempDir_DistinctSameParent(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, ".codegraph")

	tmp1, err := siblingTempDir(target)
	if err != nil {
		t.Fatalf("siblingTempDir (1): %v", err)
	}
	tmp2, err := siblingTempDir(target)
	if err != nil {
		t.Fatalf("siblingTempDir (2): %v", err)
	}

	if tmp1 == tmp2 {
		t.Fatalf("siblingTempDir returned the same path twice: %s", tmp1)
	}
	if filepath.Dir(tmp1) != parent {
		t.Fatalf("siblingTempDir(1) parent = %s, want %s", filepath.Dir(tmp1), parent)
	}
	if filepath.Dir(tmp2) != parent {
		t.Fatalf("siblingTempDir(2) parent = %s, want %s", filepath.Dir(tmp2), parent)
	}
	if !exists(tmp1) || !exists(tmp2) {
		t.Fatalf("siblingTempDir must create the directory it returns")
	}
}

// TestAtomicSwapDir_FreshTarget proves atomicSwapDir renames tmp into a
// target that does not yet exist, and tmp no longer exists afterward.
func TestAtomicSwapDir_FreshTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, ".codegraph")
	tmp := filepath.Join(parent, ".codegraph.migrate-tmp-fresh")
	writeMarker(t, tmp, "new-contents")

	if err := atomicSwapDir(tmp, target); err != nil {
		t.Fatalf("atomicSwapDir: %v", err)
	}

	if got := readMarker(t, target); got != "new-contents" {
		t.Fatalf("target marker = %q, want %q", got, "new-contents")
	}
	if exists(tmp) {
		t.Fatalf("tmp dir %s still exists after swap", tmp)
	}
}

// TestAtomicSwapDir_ExistingNonEmptyTarget proves atomicSwapDir renames the
// existing target aside, renames tmp into target, and removes the aside —
// leaving the final target holding the NEW contents with the .old gone.
func TestAtomicSwapDir_ExistingNonEmptyTarget(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, ".codegraph")
	tmp := filepath.Join(parent, ".codegraph.migrate-tmp-existing")
	writeMarker(t, target, "old-contents")
	writeMarker(t, tmp, "new-contents")

	if err := atomicSwapDir(tmp, target); err != nil {
		t.Fatalf("atomicSwapDir: %v", err)
	}

	if got := readMarker(t, target); got != "new-contents" {
		t.Fatalf("target marker = %q, want %q", got, "new-contents")
	}
	if exists(target + ".old") {
		t.Fatalf(".old path %s still exists after successful swap", target+".old")
	}
	if exists(tmp) {
		t.Fatalf("tmp dir %s still exists after swap", tmp)
	}
}

// TestAtomicSwapDir_RestoreOnFailure proves that if the rename-in step
// fails after the rename-aside step, the original is restored to target
// and the target is never left pointing at nothing.
func TestAtomicSwapDir_RestoreOnFailure(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, ".codegraph")
	writeMarker(t, target, "original-contents")

	// A tmp path that does not exist forces the rename-in step to fail
	// after the rename-aside step has already succeeded.
	bogusTmp := filepath.Join(parent, "does-not-exist-tmp-dir")

	err := atomicSwapDir(bogusTmp, target)
	if err == nil {
		t.Fatalf("atomicSwapDir with a missing tmp dir: want error, got nil")
	}

	if !exists(target) {
		t.Fatalf("target %s does not exist after failed swap — original was lost", target)
	}
	if got := readMarker(t, target); got != "original-contents" {
		t.Fatalf("target marker after restore = %q, want %q (original)", got, "original-contents")
	}
	if exists(target + ".old") {
		t.Fatalf(".old path %s should not remain after a successful restore", target+".old")
	}
}

// TestAtomicSwapDir_SameFilesystem proves the full siblingTempDir +
// atomicSwapDir happy path succeeds when tmp and target share one parent
// (never EXDEV) — builds tmp+target under one t.TempDir() root.
func TestAtomicSwapDir_SameFilesystem(t *testing.T) {
	parent := t.TempDir()
	target := filepath.Join(parent, ".codegraph")
	writeMarker(t, target, "old-contents")

	tmp, err := siblingTempDir(target)
	if err != nil {
		t.Fatalf("siblingTempDir: %v", err)
	}
	writeMarker(t, tmp, "new-contents")

	if filepath.Dir(tmp) != filepath.Dir(target) {
		t.Fatalf("tmp parent %s != target parent %s (not same filesystem)", filepath.Dir(tmp), filepath.Dir(target))
	}

	if err := atomicSwapDir(tmp, target); err != nil {
		t.Fatalf("atomicSwapDir: %v", err)
	}
	if got := readMarker(t, target); got != "new-contents" {
		t.Fatalf("target marker = %q, want %q", got, "new-contents")
	}
}

// TestCheckWritableDir proves checkWritableDir succeeds against a writable
// parent directory (the migration's own fail-fast precondition).
func TestCheckWritableDir(t *testing.T) {
	parent := t.TempDir()
	if err := checkWritableDir(parent); err != nil {
		t.Fatalf("checkWritableDir: %v", err)
	}
}
