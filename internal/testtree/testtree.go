// Package testtree writes a tree of source files for a test that indexes
// it. It holds no index step: each package that uses it indexes the tree
// its own way.
package testtree

import (
	"os"
	"path/filepath"
	"testing"
)

// T is the part of testing.TB and rapid.T that a test helper uses to
// report a failure: WriteAt, and the helpers of other packages' tests.
type T interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Write writes files under a new temporary directory and returns that
// directory. A key is a slash-separated path relative to the directory,
// and its value is the content of the file. Write makes the parent
// directories of each file.
func Write(t testing.TB, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	WriteAt(t, root, files)
	return root
}

// WriteAt writes files under root as Write does.
func WriteAt(t T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// AccountJava is the file p/Account.java for a test of data symbols. The
// class declares the field balance, which the method deposit at line 7
// reads and writes. It also declares a field and a method named size. The
// method at line 12 reads the field.
const AccountJava = `package p;

public class Account {
    private int balance;
    int size;

    public int deposit(int amount) {
        balance += amount;
        return balance;
    }

    public int size() {
        return size;
    }
}
`
