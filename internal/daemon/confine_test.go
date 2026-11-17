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
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, codegraphDirName)); err != nil {
		t.Fatal(err)
	}

	_, err := New(root, indexer.Options{Quiet: true})
	var refused *query.IndexOutsideRootError
	if !errors.As(err, &refused) {
		t.Fatalf("New = %v, want an *query.IndexOutsideRootError", err)
	}
}
