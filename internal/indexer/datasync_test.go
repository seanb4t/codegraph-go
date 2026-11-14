package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// TestDataSymbolSyncEquivalence edits the files that hold the writes and
// the declarations of data symbols, while the files that call through
// them stay unchanged. After each sync the store must equal a fresh index:
// the derived calls of the unchanged call sites follow the writes, the
// alias chain and the syntax errors of the package. A function in an
// unchanged file that reads and writes the variable keeps its write.
func TestDataSymbolSyncEquivalence(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod":           "module example.com/m\n\ngo 1.22\n",
		"daemon/daemon.go": "package daemon\n\nfunc List() {}\n\nfunc Other() {}\n",
		"cli/bind.go":      "package cli\n\nimport \"example.com/m/daemon\"\n\nvar fn = daemon.List\n\nvar alias = fn\n",
		"cli/call.go":      "package cli\n\nfunc run() { fn() }\n\nfunc runAlias() { alias() }\n",
		"cli/point.go":     "package cli\n\ntype Point struct{ X int }\n",
		"cli/get.go":       "package cli\n\nfunc (p *Point) Get() int { return p.X }\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	write := func(path, content string) {
		t.Helper()
		writeFile(t, filepath.Join(root, path), content)
	}
	remove := func(path string) {
		t.Helper()
		if err := os.Remove(filepath.Join(root, path)); err != nil {
			t.Fatal(err)
		}
	}
	runCalls := func(want bool) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		run := declaredNodes(t, r, "cli/call.go", goextract.KindFunction, "run")
		calls := len(run) == 1 && len(edgeTargets(t, r, run[0].Id, goextract.RefKindCalls)) > 0
		if calls != want {
			t.Errorf("run calls through fn: %v, want %v", calls, want)
		}
	}
	steps := []struct {
		name     string
		edit     func()
		runCalls bool
	}{
		{"add a competing write", func() {
			write("cli/conflict.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nfunc init() { fn = daemon.Other }\n")
		}, false},
		{"remove the competing write", func() { remove("cli/conflict.go") }, true},
		{"change the initializer", func() {
			write("cli/bind.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nvar fn = daemon.Other\n\nvar alias = fn\n")
		}, true},
		{"rename the variable", func() {
			write("cli/bind.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nvar fn2 = daemon.Other\n\nvar alias = fn2\n")
		}, false},
		{"restore the variable", func() {
			write("cli/bind.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nvar fn = daemon.List\n\nvar alias = fn\n")
		}, true},
		{"add a syntax error to the package", func() { write("cli/broken.go", "package cli\n\nfunc broken() {\n\tfn = (\n}\n") }, false},
		{"fix the syntax error", func() { write("cli/broken.go", "package cli\n\nfunc broken() {}\n") }, true},
		{"delete the target", func() { remove("daemon/daemon.go") }, false},
		{"restore the target", func() { write("daemon/daemon.go", "package daemon\n\nfunc List() {}\n\nfunc Other() {}\n") }, true},
		{"add a function that reads and writes the variable", func() {
			write("cli/lazy.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nfunc ensure() {\n\tif fn == nil {\n\t\tfn = daemon.List\n\t}\n}\n")
		}, true},
		{"edit only the caller", func() {
			write("cli/call.go", "package cli\n\n// edited\nfunc run() { fn() }\n\nfunc runAlias() { alias() }\n")
		}, true},
		{"move the struct", func() {
			remove("cli/point.go")
			write("cli/point2.go", "package cli\n\ntype Point struct {\n\tY int\n\tX int\n}\n")
		}, true},
	}
	for _, s := range steps {
		s.edit()
		if _, err := Sync(root, storeDir, Options{}); err != nil {
			t.Fatalf("Sync after %q: %v", s.name, err)
		}
		t.Run(s.name, func(t *testing.T) {
			runCalls(s.runCalls)
			requireSyncEqualsRun(t, root, storeDir)
		})
	}
}
