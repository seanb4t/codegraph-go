package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
	"github.com/seanb4t/codegraph-go/internal/query"
)

func TestTouchPendingRefusesSymlink(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "outside")
	data := []byte("keep me\n")
	if err := os.WriteFile(outside, data, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, staleSidecarName)); err != nil {
		t.Fatal(err)
	}

	d := &Daemon{start: query.IndexPaths{Dir: dir}}
	if err := d.touchPending(); !errors.Is(err, nofollow.ErrSymlink) {
		t.Fatalf("touchPending = %v, want nofollow.ErrSymlink", err)
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("outside file = %q, want %q", got, data)
	}
}

func TestTouchPendingCreatesSidecar(t *testing.T) {
	dir := t.TempDir()
	d := &Daemon{start: query.IndexPaths{Dir: dir}}
	if err := d.touchPending(); err != nil {
		t.Fatalf("touchPending: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, staleSidecarName)); err != nil {
		t.Fatal(err)
	}
}
