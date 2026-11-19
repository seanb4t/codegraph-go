package indexdir

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFilterSkipsThePartsOfTheIndex(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "store-real"), filepath.Join(root, Name, StoreName)); err != nil {
		t.Fatal(err)
	}
	f, err := NewFilter(root)
	if err != nil {
		t.Fatalf("NewFilter: %v", err)
	}
	for _, tc := range []struct {
		rel  string
		skip bool
	}{
		{Name, true},
		{"store-real", true},
		{"store-real" + SwapLockSuffix, true},
		{"store-real" + WriterLockSuffix, true},
		{"store-real" + ScratchSuffix + "abc", true},
		{"store-real.scratch.pending-abc", true},
		{"main.go", false},
		{"store-real2.swap", false},
		{"store-real.scratchpad", false},
		{"pkg/store-real.swap", false},
		{"store-real.swap.bak", false},
	} {
		if got := f.Skip(filepath.Join(root, tc.rel)); got != tc.skip {
			t.Errorf("Skip(%s) = %v, want %v", tc.rel, got, tc.skip)
		}
	}
	if (Filter{}).Skip(filepath.Join(root, "store-real")) {
		t.Errorf("the zero Filter skips the store")
	}
}

func TestNewFilterRefusesAnIndexOutsideTheRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(root, Name)); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFilter(root); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("NewFilter = %v, want ErrOutsideRoot", err)
	}
}
