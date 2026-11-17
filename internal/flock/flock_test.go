//go:build !windows

package flock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

func TestModesExclude(t *testing.T) {
	for _, tc := range []struct {
		name          string
		first, second Mode
		wantHeld      bool
	}{
		{"exclusive then exclusive", Exclusive, Exclusive, true},
		{"exclusive then shared", Exclusive, Shared, true},
		{"shared then exclusive", Shared, Exclusive, true},
		{"shared then shared", Shared, Shared, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lock")
			first, err := Open(path, tc.first)
			if err != nil {
				t.Fatalf("first Open: %v", err)
			}
			second, err := Open(path, tc.second)
			if err == nil {
				second.Close()
			}
			if got := IsHeld(err); got != tc.wantHeld || (err != nil) != tc.wantHeld {
				t.Fatalf("second Open: err %v, IsHeld %v, want held %v", err, got, tc.wantHeld)
			}
			// Positive assertion: the lock is free once its holder closes.
			first.Close()
			again, err := Open(path, Exclusive)
			if err != nil {
				t.Fatalf("Open after the first holder closed: %v", err)
			}
			again.Close()
		})
	}
}

func TestOpenExistingAndCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	if _, err := OpenExisting(path, Exclusive); !errors.Is(err, fs.ErrNotExist) || IsHeld(err) {
		t.Fatalf("OpenExisting on a missing file: %v, want not-exist and not held", err)
	}
	f, err := Create(path, Exclusive)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer f.Close()
	if _, err := Create(path, Exclusive); !errors.Is(err, fs.ErrExist) || IsHeld(err) {
		t.Fatalf("Create on an existing file: %v, want exist and not held", err)
	}
	if _, err := OpenExisting(path, Exclusive); !IsHeld(err) {
		t.Fatalf("OpenExisting on a held lock: %v, want held", err)
	}
}

func TestOpenRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside")
	if err := os.WriteFile(outside, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "lock")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	for name, open := range map[string]func(string, Mode) (*os.File, error){"Open": Open, "OpenExisting": OpenExisting} {
		f, err := open(link, Exclusive)
		if err == nil {
			f.Close()
		}
		if !errors.Is(err, nofollow.ErrSymlink) {
			t.Fatalf("%s = %v, want nofollow.ErrSymlink", name, err)
		}
	}
}
