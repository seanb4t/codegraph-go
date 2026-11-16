package indexer

import (
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
		{"remove the competing write", func() { removeAt(t, root, "cli/conflict.go") }, true},
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
		{"delete the target", func() { removeAt(t, root, "daemon/daemon.go") }, false},
		{"restore the target", func() { write("daemon/daemon.go", "package daemon\n\nfunc List() {}\n\nfunc Other() {}\n") }, true},
		{"add a function that reads and writes the variable", func() {
			write("cli/lazy.go", "package cli\n\nimport \"example.com/m/daemon\"\n\nfunc ensure() {\n\tif fn == nil {\n\t\tfn = daemon.List\n\t}\n}\n")
		}, true},
		{"edit only the caller", func() {
			write("cli/call.go", "package cli\n\n// edited\nfunc run() { fn() }\n\nfunc runAlias() { alias() }\n")
		}, true},
		{"move the struct", func() {
			removeAt(t, root, "cli/point.go")
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

// TestGoFieldWriteSyncEquivalence edits the Go field writes that withhold a
// call through a field: a keyed composite literal, and writes through a
// receiver of unknown type, in the package of the field and in another
// package for an exported field. A field declaration also arrives after
// a write that can reach it exists. After each sync the store must equal a fresh
// index.
func TestGoFieldWriteSyncEquivalence(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"go.mod":       "module example.com/m\n\ngo 1.22\n",
		"p/call.go":    "package p\n\nfunc run() {}\n\nfunc setup(s *S) { s.fn = run }\n\nfunc caller(s *S) { s.fn() }\n\nfunc setupE(e *E) { e.Fn = run }\n\nfunc callerE(e *E) { e.Fn() }\n",
		"p/types.go":   "package p\n\ntype S struct{ fn func() }\n",
		"ext/ext.go":   "package ext\n\nfunc Lookup() int { return 0 }\n\nfunc All() []int { return nil }\n",
		"q/unknown.go": "package q\n\nfunc poke() {}\n",
		"q/keep.go":    "package q\n\nimport \"example.com/m/ext\"\n\nvar _ = ext.Lookup\n",
	})
	storeDir := t.TempDir()
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	write := func(path, content string) {
		t.Helper()
		writeFile(t, filepath.Join(root, path), content)
	}
	callsRun := func(fn string, want bool) {
		t.Helper()
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		nodes := declaredNodes(t, r, "p/call.go", goextract.KindFunction, fn)
		got := len(nodes) == 1 && len(edgeTargets(t, r, nodes[0].Id, goextract.RefKindCalls)) > 0
		if got != want {
			t.Errorf("%s calls through its field: %v, want %v", fn, got, want)
		}
	}
	steps := []struct {
		name            string
		edit            func()
		caller, callerE bool
	}{
		{"add a keyed literal", func() { write("p/lit.go", "package p\n\nfunc build() S { return S{fn: nil} }\n") }, false, false},
		{"remove the keyed literal", func() { removeAt(t, root, "p/lit.go") }, true, false},
		{"add the exported field", func() { write("p/e.go", "package p\n\ntype E struct{ Fn func() }\n") }, true, true},
		{"add an unknown write in the package", func() {
			write("p/poke.go", "package p\n\nimport \"example.com/m/ext\"\n\nfunc poke() { x := ext.Lookup(); x.fn = nil }\n")
		}, false, true},
		{"remove the unknown write", func() { removeAt(t, root, "p/poke.go") }, true, true},
		{"add an unknown write in another package", func() {
			write("q/unknown.go", "package q\n\nimport \"example.com/m/ext\"\n\nfunc poke() { x := ext.Lookup(); x.fn = nil; x.Fn = nil }\n")
		}, true, false},
		{"remove the exported field", func() { removeAt(t, root, "p/e.go") }, true, false},
		{"add the exported field after the write", func() { write("p/e.go", "package p\n\ntype E struct{ Fn func() }\n") }, true, false},
		{"remove the unexported field and add an unknown write", func() {
			write("p/types.go", "package p\n\ntype S struct{ n int }\n")
			write("p/poke.go", "package p\n\nimport \"example.com/m/ext\"\n\nfunc poke() {\n\tfor _, x := range ext.All() {\n\t\tx.fn = nil\n\t}\n}\n")
		}, false, false},
		{"add the unexported field after the write", func() {
			removeAt(t, root, "p/types.go")
			write("p/types2.go", "package p\n\ntype S struct {\n\tn  int\n\tfn func()\n}\n")
		}, false, false},
	}
	for _, s := range steps {
		s.edit()
		if _, err := Sync(root, storeDir, Options{}); err != nil {
			t.Fatalf("Sync after %q: %v", s.name, err)
		}
		t.Run(s.name, func(t *testing.T) {
			callsRun("caller", s.caller)
			callsRun("callerE", s.callerE)
			requireSyncEqualsRun(t, root, storeDir)
		})
	}
}
