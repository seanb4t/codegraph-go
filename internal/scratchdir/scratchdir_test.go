package scratchdir

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/flock"
)

const testPrefix = "store.scratch-"

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Lstat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	return err == nil
}

// lockFiles lists every file under root whose name starts with the lock
// name, the temporary name included.
func lockFiles(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasPrefix(d.Name(), LockName) {
			found = append(found, path)
		}
		return err
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return found
}

func TestSweep(t *testing.T) {
	type lockState int
	const (
		noLock lockState = iota
		unheld
		held
		tmpOnly
	)
	for _, tc := range []struct {
		name     string
		pending  bool
		lock     lockState
		age      time.Duration
		wantGone bool
	}{
		{"final, unlocked", false, unheld, 0, true},
		{"final, no lock file", false, noLock, 0, true},
		{"pending, lock file that no process holds", true, unheld, 0, true},
		{"pending, lock held by another handle", true, held, 0, false},
		{"pending, temporary lock file only, new", true, tmpOnly, 0, false},
		{"pending, temporary lock file only, past the age limit", true, tmpOnly, PendingMaxAge + time.Second, true},
		{"pending, no lock file, under the age limit", true, noLock, PendingMaxAge - time.Second, false},
		{"pending, no lock file, past the age limit", true, noLock, PendingMaxAge + time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			name := testPrefix + "x"
			if tc.pending {
				name = PendingPrefix(testPrefix) + "x"
			}
			dir := filepath.Join(parent, name)
			other := filepath.Join(parent, "store.scratchy")
			for _, d := range []string{filepath.Join(dir, "store"), other} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
			}
			switch tc.lock {
			case unheld:
				if err := os.WriteFile(filepath.Join(dir, LockName), nil, 0o644); err != nil {
					t.Fatalf("write lock: %v", err)
				}
			case held:
				lock, err := flock.Open(filepath.Join(dir, LockName), flock.Exclusive)
				if err != nil {
					t.Fatalf("flock.Open: %v", err)
				}
				defer lock.Close()
			case tmpOnly:
				if err := os.WriteFile(filepath.Join(dir, tmpLockName), nil, 0o644); err != nil {
					t.Fatalf("write temporary lock: %v", err)
				}
			}
			if tc.age != 0 {
				then := time.Now().Add(-tc.age)
				if err := os.Chtimes(dir, then, then); err != nil {
					t.Fatalf("Chtimes: %v", err)
				}
			}
			before := lockFiles(t, parent)

			Sweep(parent, testPrefix)

			if got := !exists(t, dir); got != tc.wantGone {
				t.Errorf("directory gone = %v, want %v", got, tc.wantGone)
			}
			if !tc.wantGone && !exists(t, filepath.Join(dir, "store")) {
				t.Errorf("Sweep emptied a directory that it left in place")
			}
			if !exists(t, other) {
				t.Errorf("Sweep removed %s, which does not have the prefix", other)
			}
			// Positive assertion: Sweep created no lock file.
			for _, f := range lockFiles(t, parent) {
				created := true
				for _, b := range before {
					created = created && b != f
				}
				if created {
					t.Errorf("Sweep created the lock file %s", f)
				}
			}
		})
	}
}

func TestNewLocksBeforeItNames(t *testing.T) {
	for _, gitignore := range []bool{false, true} {
		parent := t.TempDir()
		d, err := New(parent, testPrefix, gitignore)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if filepath.Dir(d.Path()) != parent || !strings.HasPrefix(filepath.Base(d.Path()), testPrefix) {
			t.Fatalf("directory = %s, want %s/%s*", d.Path(), parent, testPrefix)
		}
		// Positive assertion: the lock file has its name, and its lock is held.
		lockPath := filepath.Join(d.Path(), LockName)
		if !exists(t, lockPath) {
			t.Fatalf("%s does not exist after New", lockPath)
		}
		if lock, err := flock.OpenExisting(lockPath, flock.Exclusive); !flock.IsHeld(err) {
			if err == nil {
				lock.Close()
			}
			t.Fatalf("a second lock on %s: %v, want held", lockPath, err)
		}
		if exists(t, filepath.Join(d.Path(), tmpLockName)) {
			t.Errorf("the temporary lock file is still in %s", d.Path())
		}
		if got := exists(t, filepath.Join(d.Path(), ".gitignore")); got != gitignore {
			t.Errorf(".gitignore exists = %v, want %v", got, gitignore)
		}
		entries, err := os.ReadDir(parent)
		if err != nil || len(entries) != 1 {
			t.Errorf("parent holds %d entries, %v; want only the directory", len(entries), err)
		}

		// A sweep by another run leaves the live directory.
		Sweep(parent, testPrefix)
		if !exists(t, d.Path()) {
			t.Fatalf("Sweep removed a directory whose owner holds its lock")
		}
		if err := d.Remove(); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if exists(t, d.Path()) {
			t.Errorf("%s still exists after Remove", d.Path())
		}
	}
}

// TestPrefixMustEndInDash checks that New refuses, and Sweep ignores, a
// prefix that does not end in "-". With such a prefix a pending name would
// start with the prefix, and Sweep with the prefix "" would take every
// directory of the parent for its own.
func TestPrefixMustEndInDash(t *testing.T) {
	for _, prefix := range []string{"", "-", "store.scratch"} {
		parent := t.TempDir()
		bystander := filepath.Join(parent, prefix+"bystander")
		if err := os.Mkdir(bystander, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		if d, err := New(parent, prefix, false); err == nil {
			_ = d.Remove()
			t.Errorf("New with the prefix %q gave no error", prefix)
		}
		Sweep(parent, prefix)
		if _, err := os.Stat(bystander); err != nil {
			t.Errorf("Sweep with the prefix %q removed a directory: %v", prefix, err)
		}
	}

	// Positive assertion: a prefix with the dash works, and Sweep removes
	// an unlocked directory of it.
	parent := t.TempDir()
	d, err := New(parent, testPrefix, false)
	if err != nil {
		t.Fatalf("New with the prefix %q: %v", testPrefix, err)
	}
	if err := d.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	Sweep(parent, testPrefix)
	if _, err := os.Stat(d.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Sweep left the unlocked directory %s: %v", d.Path(), err)
	}
}
