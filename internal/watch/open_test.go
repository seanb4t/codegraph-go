package watch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexdir"
)

func TestOpenRefusesAnIndexOutsideTheRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, indexdir.Name)); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	w, err := Open(root)
	if err == nil {
		w.Close()
	}
	if !errors.Is(err, indexdir.ErrOutsideRoot) {
		t.Fatalf("Open = %v, want indexdir.ErrOutsideRoot", err)
	}
}
