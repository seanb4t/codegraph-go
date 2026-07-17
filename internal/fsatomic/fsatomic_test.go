package fsatomic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile_NewFileGetsMode0644(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new-file.txt")

	if err := WriteFile(path, "hello"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("mode = %v, want 0644", got)
	}
}

func TestWriteFile_ExistingFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.txt")

	if err := os.WriteFile(path, []byte("old content"), 0o600); err != nil {
		t.Fatalf("setup WriteFile() error = %v", err)
	}

	if err := WriteFile(path, "new content"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %v, want 0600 (preserved)", got)
	}
}

func TestWriteFile_ByteExactContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "content.txt")

	content := "line one\nline two\nno trailing newline injected"
	if err := WriteFile(path, content); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != content {
		t.Errorf("content = %q, want %q", string(got), content)
	}
}

func TestWriteFile_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "sub", "file.txt")

	if err := WriteFile(path, "nested content"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "nested content" {
		t.Errorf("content = %q, want %q", string(got), "nested content")
	}
}

func TestWriteFile_NoTempFileLeftoverOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.txt")

	if err := WriteFile(path, "content"); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".codegraph-write-") {
			t.Errorf("leftover temp file found: %s", e.Name())
		}
	}
	if len(entries) != 1 || entries[0].Name() != "clean.txt" {
		t.Errorf("dir entries = %v, want only clean.txt", entries)
	}
}
