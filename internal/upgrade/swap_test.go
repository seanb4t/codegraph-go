package upgrade

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSwap_ReplacesTargetAtomically(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("old-binary-bytes"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := atomicSwap(target, []byte("new-binary-bytes")); err != nil {
		t.Fatalf("atomicSwap: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "new-binary-bytes" {
		t.Errorf("target contents = %q, want new-binary-bytes", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries after swap, want 1 (no leftover temp file): %v", len(entries), entries)
	}
}

func TestSwap_NotWritableTargetDirLeavesOriginalIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit writability probe is POSIX-specific")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("old-binary-bytes"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := atomicSwap(target, []byte("new-binary-bytes"))
	if err == nil {
		t.Fatal("atomicSwap: expected error on non-writable target directory, got nil")
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target after failed swap: %v", readErr)
	}
	if string(got) != "old-binary-bytes" {
		t.Errorf("target changed after failed swap: %q", got)
	}
}

func TestSwap_MissingTargetIsError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "does-not-exist")

	if err := atomicSwap(target, []byte("new-binary-bytes")); err == nil {
		t.Fatal("atomicSwap: expected error for a missing target, got nil")
	}
}
