package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

func TestWriteGitignoreHintRefusesSymlink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside")
	data := []byte("keep me\n")
	if err := os.WriteFile(outside, data, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatal(err)
	}

	if err := writeGitignoreHint(dir); !errors.Is(err, nofollow.ErrSymlink) {
		t.Fatalf("writeGitignoreHint = %v, want nofollow.ErrSymlink", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("outside file = %q, want %q", got, data)
	}
}
