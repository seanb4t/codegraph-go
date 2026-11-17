package daemon

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
)

func TestNewRefusesIndexLinkedOutside(t *testing.T) {
	for _, name := range []string{codegraphDirName, filepath.Join(codegraphDirName, storeDirName)} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), filepath.Join(root, name)); err != nil {
				t.Fatal(err)
			}

			_, err := New(root, indexer.Options{Quiet: true})
			if !errors.Is(err, query.ErrIndexOutsideRoot) {
				t.Fatalf("New = %v, want query.ErrIndexOutsideRoot", err)
			}
		})
	}
}

func TestNewUsesResolvedIndexPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".cg-real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".cg-real", filepath.Join(root, codegraphDirName)); err != nil {
		t.Fatal(err)
	}

	d, err := New(root, indexer.Options{Quiet: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	real, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(real, ".cg-real"); d.codegraphDir != want {
		t.Fatalf("codegraphDir = %q, want %q", d.codegraphDir, want)
	}
	if want := filepath.Join(real, ".cg-real", storeDirName); d.storeDir != want {
		t.Fatalf("storeDir = %q, want %q", d.storeDir, want)
	}
}
