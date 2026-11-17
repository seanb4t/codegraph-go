//go:build !windows

package nofollow

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func outsideFile(t *testing.T) (string, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "outside")
	data := []byte("keep me\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func assertUnchanged(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s = %q, want %q", path, got, want)
	}
}

func TestWriteFileRefusesSymlink(t *testing.T) {
	outside, data := outsideFile(t)
	link := filepath.Join(t.TempDir(), "file")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	err := WriteFile(link, []byte("new"), 0o644)
	var linkErr *SymlinkError
	if !errors.Is(err, ErrSymlink) || !errors.As(err, &linkErr) {
		t.Fatalf("WriteFile = %v, want a *SymlinkError", err)
	}
	if linkErr.Link != link || linkErr.Target != outside {
		t.Fatalf("SymlinkError = %+v, want Link %q, Target %q", linkErr, link, outside)
	}
	assertUnchanged(t, outside, data)
}

func TestWriteFileRefusesDanglingSymlink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing")
	link := filepath.Join(t.TempDir(), "file")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(link, []byte("new"), 0o644); !errors.Is(err, ErrSymlink) {
		t.Fatalf("WriteFile = %v, want ErrSymlink", err)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Lstat(%s) = %v, want not exist", target, err)
	}
}

func TestWriteFileWritesRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("old contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	assertUnchanged(t, path, []byte("new"))
}

func TestCheckDir(t *testing.T) {
	outside, _ := outsideFile(t)

	clean := t.TempDir()
	if err := os.WriteFile(filepath.Join(clean, "LOCK"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(clean, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(clean); err != nil {
		t.Fatalf("CheckDir(clean) = %v, want nil", err)
	}
	if err := CheckDir(filepath.Join(clean, "missing")); err != nil {
		t.Fatalf("CheckDir(missing) = %v, want nil", err)
	}

	link := filepath.Join(clean, "OPTIONS-000001")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	err := CheckDir(clean)
	var linkErr *SymlinkError
	if !errors.As(err, &linkErr) || linkErr.Link != link || linkErr.Target != outside {
		t.Fatalf("CheckDir = %v, want a *SymlinkError for %s", err, link)
	}
}

func TestOpenFileReportsLinkLoopAsSymlink(t *testing.T) {
	dir := t.TempDir()
	loop := filepath.Join(dir, "loop")
	if err := os.Symlink("loop", loop); err != nil {
		t.Fatal(err)
	}
	_, err := OpenFile(filepath.Join(loop, "file"), os.O_RDONLY, 0)
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("OpenFile = %v, want ErrSymlink", err)
	}
}

func TestCheckDirWrapsNothingForARegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckDir(path); err == nil || errors.Is(err, ErrSymlink) {
		t.Fatalf("CheckDir(file) = %v, want a non-symlink error", err)
	}
}
